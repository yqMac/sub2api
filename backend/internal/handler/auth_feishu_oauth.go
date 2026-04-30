// [bmai-fork] feishu
package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

const (
	feishuOAuthCookiePath         = "/api/v1/auth/oauth/feishu"
	feishuOAuthStateCookieName    = "feishu_oauth_state"
	feishuOAuthVerifierCookie     = "feishu_oauth_verifier"
	feishuOAuthRedirectCookie     = "feishu_oauth_redirect"
	feishuOAuthIntentCookieName   = "feishu_oauth_intent"
	feishuOAuthBindUserCookieName = "feishu_oauth_bind_user"
	feishuOAuthCookieMaxAgeSec    = 10 * 60
	feishuOAuthDefaultRedirectTo  = "/dashboard"
	feishuOAuthDefaultFrontendCB  = "/auth/feishu/callback"
)

type feishuTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type feishuTokenExchangeError struct {
	StatusCode          int
	ProviderError       string
	ProviderDescription string
	Body                string
}

func (e *feishuTokenExchangeError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("token exchange status=%d", e.StatusCode)}
	if strings.TrimSpace(e.ProviderError) != "" {
		parts = append(parts, "error="+strings.TrimSpace(e.ProviderError))
	}
	if strings.TrimSpace(e.ProviderDescription) != "" {
		parts = append(parts, "error_description="+strings.TrimSpace(e.ProviderDescription))
	}
	return strings.Join(parts, " ")
}

type feishuUserInfoClaims struct {
	Email           string
	EnterpriseEmail string
	Username        string
	Subject         string
	TenantKey       string
	OpenID          string
	UnionID         string
	DisplayName     string
	AvatarURL       string
}

func (h *AuthHandler) FeishuOAuthStart(c *gin.Context) {
	cfg, err := h.getFeishuOAuthConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	redirectTo := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirectTo == "" {
		redirectTo = feishuOAuthDefaultRedirectTo
	}

	browserSessionKey, err := generateOAuthPendingBrowserSession()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BROWSER_SESSION_GEN_FAILED", "failed to generate oauth browser session").WithCause(err))
		return
	}

	secureCookie := isRequestHTTPS(c)
	feishuSetCookie(c, feishuOAuthStateCookieName, encodeCookieValue(state), feishuOAuthCookieMaxAgeSec, secureCookie)
	feishuSetCookie(c, feishuOAuthRedirectCookie, encodeCookieValue(redirectTo), feishuOAuthCookieMaxAgeSec, secureCookie)
	intent := normalizeOAuthIntent(c.Query("intent"))
	feishuSetCookie(c, feishuOAuthIntentCookieName, encodeCookieValue(intent), feishuOAuthCookieMaxAgeSec, secureCookie)
	setOAuthPendingBrowserCookie(c, browserSessionKey, secureCookie)
	clearOAuthPendingSessionCookie(c, secureCookie)
	if intent == oauthIntentBindCurrentUser {
		bindCookieValue, err := h.buildOAuthBindUserCookieFromContext(c)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		feishuSetCookie(c, feishuOAuthBindUserCookieName, encodeCookieValue(bindCookieValue), feishuOAuthCookieMaxAgeSec, secureCookie)
	} else {
		feishuClearCookie(c, feishuOAuthBindUserCookieName, secureCookie)
	}

	codeChallenge := ""
	if cfg.UsePKCE {
		verifier, err := oauth.GenerateCodeVerifier()
		if err != nil {
			response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_PKCE_GEN_FAILED", "failed to generate pkce verifier").WithCause(err))
			return
		}
		codeChallenge = oauth.GenerateCodeChallenge(verifier)
		feishuSetCookie(c, feishuOAuthVerifierCookie, encodeCookieValue(verifier), feishuOAuthCookieMaxAgeSec, secureCookie)
	}

	redirectURI := strings.TrimSpace(cfg.RedirectURL)
	if redirectURI == "" {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured"))
		return
	}

	authURL, err := buildFeishuAuthorizeURL(cfg, state, codeChallenge, redirectURI)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BUILD_URL_FAILED", "failed to build oauth authorization url").WithCause(err))
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

func (h *AuthHandler) FeishuOAuthCallback(c *gin.Context) {
	cfg, cfgErr := h.getFeishuOAuthConfig(c.Request.Context())
	if cfgErr != nil {
		response.ErrorFrom(c, cfgErr)
		return
	}

	frontendCallback := strings.TrimSpace(cfg.FrontendRedirectURL)
	if frontendCallback == "" {
		frontendCallback = feishuOAuthDefaultFrontendCB
	}

	if providerErr := strings.TrimSpace(c.Query("error")); providerErr != "" {
		redirectOAuthError(c, frontendCallback, "provider_error", providerErr, c.Query("error_description"))
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		redirectOAuthError(c, frontendCallback, "missing_params", "missing code/state", "")
		return
	}

	secureCookie := isRequestHTTPS(c)
	defer func() {
		feishuClearCookie(c, feishuOAuthStateCookieName, secureCookie)
		feishuClearCookie(c, feishuOAuthVerifierCookie, secureCookie)
		feishuClearCookie(c, feishuOAuthRedirectCookie, secureCookie)
		feishuClearCookie(c, feishuOAuthIntentCookieName, secureCookie)
		feishuClearCookie(c, feishuOAuthBindUserCookieName, secureCookie)
	}()

	expectedState, err := readCookieDecoded(c, feishuOAuthStateCookieName)
	if err != nil || expectedState == "" || state != expectedState {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}

	redirectTo, _ := readCookieDecoded(c, feishuOAuthRedirectCookie)
	redirectTo = sanitizeFrontendRedirectPath(redirectTo)
	if redirectTo == "" {
		redirectTo = feishuOAuthDefaultRedirectTo
	}
	browserSessionKey, _ := readOAuthPendingBrowserCookie(c)
	if strings.TrimSpace(browserSessionKey) == "" {
		redirectOAuthError(c, frontendCallback, "missing_browser_session", "missing oauth browser session", "")
		return
	}
	intent, _ := readCookieDecoded(c, feishuOAuthIntentCookieName)
	intent = normalizeOAuthIntent(intent)

	codeVerifier := ""
	if cfg.UsePKCE {
		codeVerifier, _ = readCookieDecoded(c, feishuOAuthVerifierCookie)
		if codeVerifier == "" {
			redirectOAuthError(c, frontendCallback, "missing_verifier", "missing pkce verifier", "")
			return
		}
	}

	redirectURI := strings.TrimSpace(cfg.RedirectURL)
	if redirectURI == "" {
		redirectOAuthError(c, frontendCallback, "config_error", "oauth redirect url not configured", "")
		return
	}

	tokenResp, err := feishuExchangeCode(c.Request.Context(), cfg, code, redirectURI, codeVerifier)
	if err != nil {
		description := ""
		var exchangeErr *feishuTokenExchangeError
		if errors.As(err, &exchangeErr) && exchangeErr != nil {
			log.Printf(
				"[Feishu OAuth] token exchange failed: status=%d provider_error=%q provider_description=%q body=%s",
				exchangeErr.StatusCode,
				exchangeErr.ProviderError,
				exchangeErr.ProviderDescription,
				truncateLogValue(exchangeErr.Body, 2048),
			)
			description = exchangeErr.Error()
		} else {
			log.Printf("[Feishu OAuth] token exchange failed: %v", err)
			description = err.Error()
		}
		redirectOAuthError(c, frontendCallback, "token_exchange_failed", "failed to exchange oauth code", singleLine(description))
		return
	}

	userClaims, err := feishuFetchUserInfo(c.Request.Context(), cfg, tokenResp)
	if err != nil {
		log.Printf("[Feishu OAuth] userinfo fetch failed: %v", err)
		redirectOAuthError(c, frontendCallback, "userinfo_failed", "failed to fetch user info", "")
		return
	}
	log.Printf("[Feishu OAuth] userinfo: subject=%s email=%q enterprise_email=%q tenant_key=%s display_name=%q",
		userClaims.Subject, userClaims.Email, userClaims.EnterpriseEmail, userClaims.TenantKey, userClaims.DisplayName)

	if !service.IsFeishuTenantAllowed(cfg.AllowedTenantKeys, userClaims.TenantKey) {
		redirectOAuthError(c, frontendCallback, "tenant_not_allowed", "tenant is not allowed", "")
		return
	}

	subject := strings.TrimSpace(userClaims.Subject)
	compatEmail := strings.TrimSpace(userClaims.EnterpriseEmail)
	if compatEmail == "" && !cfg.RequireEnterpriseEmail {
		compatEmail = strings.TrimSpace(userClaims.Email)
	}
	syntheticEmail := service.FeishuSyntheticEmail(subject)
	resolvedEmail := compatEmail
	if resolvedEmail == "" {
		resolvedEmail = syntheticEmail
	}

	identityKey := service.PendingAuthIdentityKey{
		ProviderType:    "feishu",
		ProviderKey:     "feishu",
		ProviderSubject: subject,
	}
	upstreamClaims := map[string]any{
		"email":                  resolvedEmail,
		"username":               strings.TrimSpace(userClaims.Username),
		"subject":                subject,
		"tenant_key":             strings.TrimSpace(userClaims.TenantKey),
		"open_id":                strings.TrimSpace(userClaims.OpenID),
		"union_id":               strings.TrimSpace(userClaims.UnionID),
		"suggested_display_name": strings.TrimSpace(userClaims.DisplayName),
		"suggested_avatar_url":   strings.TrimSpace(userClaims.AvatarURL),
	}
	if compatEmail != "" {
		upstreamClaims["compat_email"] = compatEmail
	}

	if intent == oauthIntentBindCurrentUser {
		targetUserID, err := h.readOAuthBindUserIDFromCookie(c, feishuOAuthBindUserCookieName)
		if err != nil {
			redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth bind target", "")
			return
		}
		if err := h.createOAuthPendingSession(c, oauthPendingSessionPayload{
			Intent:                 oauthIntentBindCurrentUser,
			Identity:               identityKey,
			TargetUserID:           &targetUserID,
			ResolvedEmail:          resolvedEmail,
			RedirectTo:             redirectTo,
			BrowserSessionKey:      browserSessionKey,
			UpstreamIdentityClaims: upstreamClaims,
			CompletionResponse: map[string]any{
				"redirect": redirectTo,
			},
		}); err != nil {
			redirectOAuthError(c, frontendCallback, "session_error", "failed to continue oauth bind", "")
			return
		}
		redirectToFrontendCallback(c, frontendCallback)
		return
	}

	existingIdentityUser, err := h.findOAuthIdentityUser(c.Request.Context(), identityKey)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}
	if existingIdentityUser != nil {
		if cfg.AutoBindOrCreate { // [bmai-fork] feishu
			if err := h.completeFeishuAutoLogin(c, frontendCallback, redirectTo, existingIdentityUser.ID, identityKey, upstreamClaims); err != nil {
				redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
				return
			}
			return
		}
		if err := h.createOAuthPendingSession(c, oauthPendingSessionPayload{
			Intent:                 oauthIntentLogin,
			Identity:               identityKey,
			TargetUserID:           &existingIdentityUser.ID,
			ResolvedEmail:          existingIdentityUser.Email,
			RedirectTo:             redirectTo,
			BrowserSessionKey:      browserSessionKey,
			UpstreamIdentityClaims: upstreamClaims,
			CompletionResponse: map[string]any{
				"redirect": redirectTo,
			},
		}); err != nil {
			redirectOAuthError(c, frontendCallback, "session_error", "failed to continue oauth login", "")
			return
		}
		redirectToFrontendCallback(c, frontendCallback)
		return
	}

	compatEmailUser, err := h.findFeishuCompatEmailUser(c.Request.Context(), compatEmail)
	if err != nil {
		redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}
	if cfg.AutoBindOrCreate { // [bmai-fork] feishu
		if compatEmailUser != nil {
			if err := h.completeFeishuAutoLogin(c, frontendCallback, redirectTo, compatEmailUser.ID, identityKey, upstreamClaims); err != nil {
				redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
				return
			}
			return
		}
		if err := h.completeFeishuAutoCreate(c, frontendCallback, redirectTo, resolvedEmail, strings.TrimSpace(userClaims.Username), identityKey, upstreamClaims); err != nil {
			redirectOAuthError(c, frontendCallback, "session_error", infraerrors.Reason(err), infraerrors.Message(err))
			return
		}
		return
	}

	if err := h.createFeishuOAuthChoicePendingSession(
		c,
		identityKey,
		resolvedEmail,
		resolvedEmail,
		redirectTo,
		browserSessionKey,
		upstreamClaims,
		compatEmail,
		compatEmailUser,
		h.isForceEmailOnThirdPartySignup(c.Request.Context()),
	); err != nil {
		redirectOAuthError(c, frontendCallback, "session_error", "failed to continue oauth login", "")
		return
	}
	redirectToFrontendCallback(c, frontendCallback)
}

func (h *AuthHandler) findFeishuCompatEmailUser(ctx context.Context, email string) (*dbent.User, error) {
	client := h.entClient()
	if client == nil {
		return nil, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" ||
		strings.HasSuffix(email, service.LinuxDoConnectSyntheticEmailDomain) ||
		strings.HasSuffix(email, service.OIDCConnectSyntheticEmailDomain) ||
		strings.HasSuffix(email, service.WeChatConnectSyntheticEmailDomain) ||
		strings.HasSuffix(email, service.FeishuConnectSyntheticEmailDomain) {
		return nil, nil
	}

	userEntity, err := findUserByNormalizedEmail(ctx, client, email)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			return nil, nil
		}
		return nil, infraerrors.InternalServer("COMPAT_EMAIL_LOOKUP_FAILED", "failed to look up compat email user").WithCause(err)
	}
	return userEntity, nil
}

func (h *AuthHandler) createFeishuOAuthChoicePendingSession(
	c *gin.Context,
	identity service.PendingAuthIdentityKey,
	suggestedEmail string,
	resolvedEmail string,
	redirectTo string,
	browserSessionKey string,
	upstreamClaims map[string]any,
	compatEmail string,
	compatEmailUser *dbent.User,
	forceEmailOnSignup bool,
) error {
	suggestionEmail := strings.TrimSpace(suggestedEmail)
	canonicalEmail := strings.TrimSpace(resolvedEmail)
	if suggestionEmail == "" {
		suggestionEmail = canonicalEmail
	}

	completionResponse := map[string]any{
		"step":                      oauthPendingChoiceStep,
		"adoption_required":         true,
		"redirect":                  strings.TrimSpace(redirectTo),
		"email":                     suggestionEmail,
		"resolved_email":            canonicalEmail,
		"existing_account_email":    "",
		"existing_account_bindable": false,
		"create_account_allowed":    true,
		"force_email_on_signup":     forceEmailOnSignup,
		"choice_reason":             "third_party_signup",
	}
	if strings.TrimSpace(compatEmail) != "" {
		completionResponse["compat_email"] = strings.TrimSpace(compatEmail)
	}
	resolvedChoiceEmail := suggestionEmail
	if compatEmailUser != nil {
		completionResponse["email"] = strings.TrimSpace(compatEmailUser.Email)
		completionResponse["existing_account_email"] = strings.TrimSpace(compatEmailUser.Email)
		completionResponse["existing_account_bindable"] = true
		completionResponse["choice_reason"] = "compat_email_match"
		resolvedChoiceEmail = strings.TrimSpace(compatEmailUser.Email)
	}
	if forceEmailOnSignup && compatEmailUser == nil {
		completionResponse["choice_reason"] = "force_email_on_signup"
	}

	var targetUserID *int64
	if compatEmailUser != nil && compatEmailUser.ID > 0 {
		targetUserID = &compatEmailUser.ID
	}

	return h.createOAuthPendingSession(c, oauthPendingSessionPayload{
		Intent:                 oauthIntentLogin,
		Identity:               identity,
		TargetUserID:           targetUserID,
		ResolvedEmail:          resolvedChoiceEmail,
		RedirectTo:             redirectTo,
		BrowserSessionKey:      browserSessionKey,
		UpstreamIdentityClaims: upstreamClaims,
		CompletionResponse:     completionResponse,
	})
}

func (h *AuthHandler) completeFeishuAutoLogin(
	c *gin.Context,
	frontendCallback string,
	redirectTo string,
	userID int64,
	identity service.PendingAuthIdentityKey,
	upstreamClaims map[string]any,
) error {
	if err := h.ensureFeishuRuntimeIdentityBinding(c.Request.Context(), userID, identity, upstreamClaims); err != nil {
		return err
	}
	if h.userService == nil || h.authService == nil {
		return infraerrors.ServiceUnavailable("AUTH_NOT_READY", "authentication service is not ready")
	}
	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive() {
		return infraerrors.Forbidden("ACCOUNT_DISABLED", "account is disabled")
	}
	tokenPair, err := h.authService.GenerateTokenPair(c.Request.Context(), user, "")
	if err != nil {
		return infraerrors.InternalServer("TOKEN_GEN_FAILED", "failed to generate token pair").WithCause(err)
	}
	h.authService.RecordSuccessfulLogin(c.Request.Context(), user.ID)
	feishuRedirectWithToken(c, frontendCallback, tokenPair, redirectTo)
	return nil
}

func (h *AuthHandler) completeFeishuAutoCreate(
	c *gin.Context,
	frontendCallback string,
	redirectTo string,
	email string,
	username string,
	identity service.PendingAuthIdentityKey,
	upstreamClaims map[string]any,
) error {
	if err := h.ensureBackendModeAllowsNewUserLogin(c.Request.Context()); err != nil {
		return err
	}
	if h.authService == nil {
		return infraerrors.ServiceUnavailable("AUTH_NOT_READY", "authentication service is not ready")
	}
	tokenPair, user, err := h.authService.LoginOrRegisterOAuthWithTokenPair(c.Request.Context(), email, username, "", "")
	if err != nil {
		return err
	}
	if user == nil {
		return infraerrors.InternalServer("USER_CREATE_FAILED", "failed to resolve oauth user")
	}
	if err := h.ensureFeishuRuntimeIdentityBinding(c.Request.Context(), user.ID, identity, upstreamClaims); err != nil {
		return err
	}
	h.authService.RecordSuccessfulLogin(c.Request.Context(), user.ID)
	feishuRedirectWithToken(c, frontendCallback, tokenPair, redirectTo)
	return nil
}

func (h *AuthHandler) ensureFeishuRuntimeIdentityBinding(
	ctx context.Context,
	userID int64,
	identity service.PendingAuthIdentityKey,
	upstreamClaims map[string]any,
) error {
	client := h.entClient()
	if client == nil {
		return infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready")
	}
	tx, err := client.Tx(ctx)
	if err != nil {
		return infraerrors.InternalServer("AUTH_IDENTITY_BIND_FAILED", "failed to begin feishu identity bind transaction").WithCause(err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = ensurePendingOAuthIdentityForUser(dbent.NewTxContext(ctx, tx), tx, &dbent.PendingAuthSession{
		ProviderType:           strings.TrimSpace(identity.ProviderType),
		ProviderKey:            strings.TrimSpace(identity.ProviderKey),
		ProviderSubject:        strings.TrimSpace(identity.ProviderSubject),
		UpstreamIdentityClaims: cloneOAuthMetadata(upstreamClaims),
	}, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func feishuRedirectWithToken(c *gin.Context, frontendCallback string, tokenPair *service.TokenPair, redirectTo string) {
	fragment := url.Values{}
	if tokenPair != nil {
		fragment.Set("access_token", strings.TrimSpace(tokenPair.AccessToken))
		if strings.TrimSpace(tokenPair.RefreshToken) != "" {
			fragment.Set("refresh_token", strings.TrimSpace(tokenPair.RefreshToken))
		}
		if tokenPair.ExpiresIn > 0 {
			fragment.Set("expires_in", strconv.Itoa(tokenPair.ExpiresIn))
		}
		fragment.Set("token_type", "Bearer")
	}
	if sanitized := sanitizeFrontendRedirectPath(strings.TrimSpace(redirectTo)); sanitized != "" {
		fragment.Set("redirect", sanitized)
	}
	redirectWithFragment(c, frontendCallback, fragment)
}

type completeFeishuOAuthRequest struct {
	InvitationCode   string `json:"invitation_code" binding:"required"`
	AffCode          string `json:"aff_code,omitempty"`
	AdoptDisplayName *bool  `json:"adopt_display_name,omitempty"`
	AdoptAvatar      *bool  `json:"adopt_avatar,omitempty"`
}

func (h *AuthHandler) CompleteFeishuOAuthRegistration(c *gin.Context) {
	var req completeFeishuOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	secureCookie := isRequestHTTPS(c)
	sessionToken, err := readOAuthPendingSessionCookie(c)
	if err != nil {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
		response.ErrorFrom(c, service.ErrPendingAuthSessionNotFound)
		return
	}
	browserSessionKey, err := readOAuthPendingBrowserCookie(c)
	if err != nil {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
		response.ErrorFrom(c, service.ErrPendingAuthBrowserMismatch)
		return
	}
	pendingSvc, err := h.pendingIdentityService()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	session, err := pendingSvc.GetBrowserSession(c.Request.Context(), sessionToken, browserSessionKey)
	if err != nil {
		clearOAuthPendingSessionCookie(c, secureCookie)
		clearOAuthPendingBrowserCookie(c, secureCookie)
		response.ErrorFrom(c, err)
		return
	}
	if err := ensurePendingOAuthCompleteRegistrationSession(session); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if updatedSession, handled, err := h.legacyCompleteRegistrationSessionStatus(c, session); err != nil {
		response.ErrorFrom(c, err)
		return
	} else if handled {
		c.JSON(http.StatusOK, buildPendingOAuthSessionStatusPayload(updatedSession))
		return
	} else {
		session = updatedSession
	}
	if err := h.ensureBackendModeAllowsNewUserLogin(c.Request.Context()); err != nil {
		response.ErrorFrom(c, err)
		return
	}

	email := strings.TrimSpace(session.ResolvedEmail)
	username := pendingSessionStringValue(session.UpstreamIdentityClaims, "username")
	if email == "" || username == "" {
		response.ErrorFrom(c, infraerrors.BadRequest("PENDING_AUTH_SESSION_INVALID", "pending auth registration context is invalid"))
		return
	}

	client := h.entClient()
	if client == nil {
		response.ErrorFrom(c, infraerrors.ServiceUnavailable("PENDING_AUTH_NOT_READY", "pending auth service is not ready"))
		return
	}
	if err := ensurePendingOAuthRegistrationIdentityAvailable(c.Request.Context(), client, session); err != nil {
		respondPendingOAuthBindingApplyError(c, err)
		return
	}
	decision, err := h.ensurePendingOAuthAdoptionDecision(c, session.ID, oauthAdoptionDecisionRequest{
		AdoptDisplayName: req.AdoptDisplayName,
		AdoptAvatar:      req.AdoptAvatar,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	tokenPair, user, err := h.authService.LoginOrRegisterOAuthWithTokenPair(c.Request.Context(), email, username, req.InvitationCode, req.AffCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if err := applyPendingOAuthAdoptionAndConsumeSession(c.Request.Context(), client, h.authService, h.userService, session, decision, user.ID); err != nil {
		respondPendingOAuthBindingApplyError(c, err)
		return
	}
	h.authService.RecordSuccessfulLogin(c.Request.Context(), user.ID)
	clearOAuthPendingSessionCookie(c, secureCookie)
	clearOAuthPendingBrowserCookie(c, secureCookie)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) getFeishuOAuthConfig(ctx context.Context) (config.FeishuConnectConfig, error) {
	if h != nil && h.settingSvc != nil {
		type feishuOAuthConfigProvider interface {
			GetFeishuConnectOAuthConfig(context.Context) (config.FeishuConnectConfig, error)
		}
		if provider, ok := any(h.settingSvc).(feishuOAuthConfigProvider); ok {
			return provider.GetFeishuConnectOAuthConfig(ctx)
		}
	}
	if h == nil || h.cfg == nil {
		return config.FeishuConnectConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	if !h.cfg.Feishu.Enabled {
		return config.FeishuConnectConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	return h.cfg.Feishu, nil
}

func feishuExchangeCode(
	ctx context.Context,
	cfg config.FeishuConnectConfig,
	code string,
	redirectURI string,
	codeVerifier string,
) (*feishuTokenResponse, error) {
	client := req.C().SetTimeout(30 * time.Second)

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	if strings.TrimSpace(codeVerifier) != "" {
		form.Set("code_verifier", codeVerifier)
	}

	r := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json")

	switch strings.ToLower(strings.TrimSpace(cfg.TokenAuthMethod)) {
	case "", "client_secret_post":
		form.Set("client_secret", cfg.ClientSecret)
	case "client_secret_basic":
		r.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)
	case "none":
	default:
		return nil, fmt.Errorf("unsupported token_auth_method: %s", cfg.TokenAuthMethod)
	}

	resp, err := r.SetFormDataFromValues(form).Post(cfg.TokenURL)
	if err != nil {
		return nil, fmt.Errorf("request token: %w", err)
	}
	body := strings.TrimSpace(resp.String())
	if !resp.IsSuccessState() {
		providerErr, providerDesc := parseOAuthProviderError(body)
		return nil, &feishuTokenExchangeError{
			StatusCode:          resp.StatusCode,
			ProviderError:       providerErr,
			ProviderDescription: providerDesc,
			Body:                body,
		}
	}

	tokenResp, ok := parseFeishuTokenResponse(body)
	if !ok || strings.TrimSpace(tokenResp.AccessToken) == "" {
		return nil, &feishuTokenExchangeError{StatusCode: resp.StatusCode, Body: body}
	}
	if strings.TrimSpace(tokenResp.TokenType) == "" {
		tokenResp.TokenType = "Bearer"
	}
	return tokenResp, nil
}

func parseFeishuTokenResponse(body string) (*feishuTokenResponse, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, false
	}

	accessToken := strings.TrimSpace(getGJSON(body, "access_token"))
	if accessToken != "" {
		tokenType := strings.TrimSpace(getGJSON(body, "token_type"))
		refreshToken := strings.TrimSpace(getGJSON(body, "refresh_token"))
		scope := strings.TrimSpace(getGJSON(body, "scope"))
		expiresIn := gjson.Get(body, "expires_in").Int()
		return &feishuTokenResponse{
			AccessToken:  accessToken,
			TokenType:    tokenType,
			ExpiresIn:    expiresIn,
			RefreshToken: refreshToken,
			Scope:        scope,
		}, true
	}

	values, err := url.ParseQuery(body)
	if err != nil {
		return nil, false
	}
	accessToken = strings.TrimSpace(values.Get("access_token"))
	if accessToken == "" {
		return nil, false
	}
	expiresIn := int64(0)
	if raw := strings.TrimSpace(values.Get("expires_in")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			expiresIn = v
		}
	}
	return &feishuTokenResponse{
		AccessToken:  accessToken,
		TokenType:    strings.TrimSpace(values.Get("token_type")),
		ExpiresIn:    expiresIn,
		RefreshToken: strings.TrimSpace(values.Get("refresh_token")),
		Scope:        strings.TrimSpace(values.Get("scope")),
	}, true
}

func feishuFetchUserInfo(
	ctx context.Context,
	cfg config.FeishuConnectConfig,
	token *feishuTokenResponse,
) (*feishuUserInfoClaims, error) {
	client := req.C().SetTimeout(30 * time.Second)
	authorization, err := buildBearerAuthorization(token.TokenType, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token for userinfo request: %w", err)
	}

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", authorization).
		Get(cfg.UserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("request userinfo: %w", err)
	}
	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("userinfo status=%d", resp.StatusCode)
	}

	return feishuParseUserInfo(resp.String(), cfg)
}

func feishuParseUserInfo(body string, cfg config.FeishuConnectConfig) (*feishuUserInfoClaims, error) {
	claims := &feishuUserInfoClaims{}
	claims.Subject = firstNonEmpty(
		getGJSON(body, cfg.UserInfoIDPath),
		getGJSON(body, "sub"),
		getGJSON(body, "open_id"),
		getGJSON(body, "user_id"),
	)
	claims.Subject = strings.TrimSpace(claims.Subject)
	if claims.Subject == "" {
		return nil, errors.New("userinfo missing id field")
	}
	if !isSafeFeishuSubject(claims.Subject) {
		return nil, errors.New("userinfo returned invalid id field")
	}

	claims.Email = strings.TrimSpace(firstNonEmpty(
		getGJSON(body, cfg.UserInfoEmailPath),
		getGJSON(body, "email"),
	))
	claims.EnterpriseEmail = strings.TrimSpace(getGJSON(body, "enterprise_email"))
	claims.TenantKey = strings.TrimSpace(getGJSON(body, "tenant_key"))
	claims.OpenID = strings.TrimSpace(firstNonEmpty(getGJSON(body, "open_id"), claims.Subject))
	claims.UnionID = strings.TrimSpace(getGJSON(body, "union_id"))
	claims.DisplayName = strings.TrimSpace(firstNonEmpty(getGJSON(body, "name"), getGJSON(body, "en_name")))
	claims.AvatarURL = strings.TrimSpace(firstNonEmpty(getGJSON(body, "picture"), getGJSON(body, "avatar_url"), getGJSON(body, "avatar_big")))
	claims.Username = strings.TrimSpace(firstNonEmpty(
		getGJSON(body, cfg.UserInfoUsernamePath),
		claims.DisplayName,
	))
	if claims.Username == "" {
		claims.Username = "feishu_" + claims.Subject
	}

	syntheticEmail := service.FeishuSyntheticEmail(claims.Subject)
	if cfg.RequireEnterpriseEmail {
		claims.Email = firstNonEmpty(claims.EnterpriseEmail, syntheticEmail)
	} else {
		claims.Email = firstNonEmpty(claims.EnterpriseEmail, claims.Email, syntheticEmail)
	}

	return claims, nil
}

func buildFeishuAuthorizeURL(cfg config.FeishuConnectConfig, state string, codeChallenge string, redirectURI string) (string, error) {
	u, err := url.Parse(cfg.AuthorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize_url: %w", err)
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	if strings.TrimSpace(cfg.Scopes) != "" {
		q.Set("scope", cfg.Scopes)
	}
	q.Set("state", state)
	if strings.TrimSpace(codeChallenge) != "" {
		q.Set("code_challenge", codeChallenge)
		q.Set("code_challenge_method", "S256")
	}

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func feishuSetCookie(c *gin.Context, name, value string, maxAgeSec int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     feishuOAuthCookiePath,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func feishuClearCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     feishuOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func isSafeFeishuSubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	if subject == "" || len(subject) > 128 {
		return false
	}
	for _, r := range subject {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_' || r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}
