// [bmai-fork] gateway audit helpers — wire audit capture into the request path
package handler

import (
	"bytes"
	"context"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// auditAcquireBuffer returns a capture buffer when audit is enabled, or nil.
// The buffer is automatically attached to the context so streaming code can tee into it.
func (h *GatewayHandler) auditAcquireBuffer(ctx context.Context) (*service.AuditCaptureBuffer, context.Context) {
	if h.auditLogService == nil || !h.auditLogService.Enabled() {
		return nil, ctx
	}
	buf := service.AcquireAuditCaptureBuffer(h.auditLogService.MaxResponseBytes())
	return buf, service.WithAuditCaptureBuffer(ctx, buf)
}

// auditReleaseBuffer returns the buffer to the pool. Safe to call with nil.
func (h *GatewayHandler) auditReleaseBuffer(buf *service.AuditCaptureBuffer) {
	service.ReleaseAuditCaptureBuffer(buf)
}

// auditSubmitInput captures everything needed to build an AuditLog asynchronously.
type auditSubmitInput struct {
	RequestID        string
	UserID           int64
	APIKeyID         int64
	AccountID        int64
	OrganizationID   *int64
	DepartmentID     *int64
	DepartmentPath   string
	RequestBody      []byte
	ResponseBody     []byte
	ResponseTotal    int
	ResponseTrunc    bool
	Model            string
	Platform         string
	Endpoint         string
	Stream           bool
	DurationMs       int
	InputTokens      int
	OutputTokens     int
	StatusCode       int
	HasTools         bool
	HasThinking      bool
	HasReasoning     bool
	BillingMode      string
}

// submitAuditLog builds and submits an AuditLog. No-op when audit is disabled.
func (h *GatewayHandler) submitAuditLog(in auditSubmitInput) {
	if h.auditLogService == nil || !h.auditLogService.Enabled() {
		return
	}
	maxReq := h.auditLogService.MaxRequestBytes()

	reqPreview, reqTrunc := truncatePreview(in.RequestBody, maxReq)
	respPreview := string(in.ResponseBody)

	contentType := service.ClassifyContentType(service.AuditClassifyInput{
		Endpoint:     in.Endpoint,
		Model:        in.Model,
		BillingMode:  in.BillingMode,
		HasTools:     in.HasTools,
		HasThinking:  in.HasThinking,
		HasReasoning: in.HasReasoning,
	})
	if h.auditLogService.ClassifyResponse() && len(in.ResponseBody) > 0 {
		contentType = service.RefineContentType(contentType, in.ResponseBody)
	}

	log := &domain.AuditLog{
		RequestID:         in.RequestID,
		UserID:            in.UserID,
		APIKeyID:          in.APIKeyID,
		AccountID:         in.AccountID,
		OrganizationID:    in.OrganizationID,
		DepartmentID:      in.DepartmentID,
		DepartmentPath:    in.DepartmentPath,
		ContentType:       contentType,
		RequestPreview:    reqPreview,
		RequestBytes:      len(in.RequestBody),
		RequestTruncated:  reqTrunc,
		ResponsePreview:   respPreview,
		ResponseBytes:     in.ResponseTotal,
		ResponseTruncated: in.ResponseTrunc,
		Model:             in.Model,
		Platform:          in.Platform,
		Endpoint:          in.Endpoint,
		Stream:            in.Stream,
		DurationMs:        in.DurationMs,
		InputTokens:       in.InputTokens,
		OutputTokens:      in.OutputTokens,
		StatusCode:        in.StatusCode,
		CreatedAt:         time.Now(),
	}
	h.auditLogService.Submit(log)
}

func truncatePreview(b []byte, max int) (string, bool) {
	if max <= 0 || len(b) <= max {
		return string(b), false
	}
	return strings.ToValidUTF8(string(b[:max]), ""), true
}

// detectHasTools is a lightweight check for tools/tool_choice presence in the request body.
// Avoids re-parsing JSON; tolerates false positives — classification is a hint, not security.
func detectHasTools(body []byte) bool {
	return bytes.Contains(body, []byte(`"tools"`)) || bytes.Contains(body, []byte(`"tool_choice"`))
}

// submitAuditFromMessagesContext bridges Messages handler state into submitAuditLog.
func (h *GatewayHandler) submitAuditFromMessagesContext(
	parsedReq *service.ParsedRequest,
	account *service.Account,
	apiKey *service.APIKey,
	result *service.ForwardResult,
	auditBuf *service.AuditCaptureBuffer,
	requestBody []byte,
	inboundEndpoint string,
) {
	if h.auditLogService == nil || !h.auditLogService.Enabled() || result == nil {
		return
	}

	var (
		respBody  []byte
		respTotal int
		respTrunc bool
	)
	if len(result.ResponseBodyPreview) > 0 {
		respBody = result.ResponseBodyPreview
		respTotal = result.ResponseBodyBytes
		respTrunc = result.ResponseBodyBytes > len(result.ResponseBodyPreview)
	} else if auditBuf != nil {
		respBody = auditBuf.Bytes()
		respTotal = auditBuf.Total
		respTrunc = auditBuf.Truncated()
	}

	in := auditSubmitInput{
		RequestID:     result.RequestID,
		UserID:        apiKey.UserID,
		APIKeyID:      apiKey.ID,
		AccountID:     account.ID,
		RequestBody:   requestBody,
		ResponseBody:  respBody,
		ResponseTotal: respTotal,
		ResponseTrunc: respTrunc,
		Model:         result.Model,
		Platform:      account.Platform,
		Endpoint:      inboundEndpoint,
		Stream:        result.Stream,
		DurationMs:    int(result.Duration.Milliseconds()),
		InputTokens:   int(result.Usage.InputTokens),
		OutputTokens:  int(result.Usage.OutputTokens),
		StatusCode:    200,
		HasTools:      detectHasTools(requestBody),
		HasThinking:   parsedReq.ThinkingEnabled,
		HasReasoning:  parsedReq.OutputEffort != "",
	}
	if apiKey.User != nil {
		in.OrganizationID = apiKey.User.OrganizationID
		in.DepartmentID = apiKey.User.PrimaryDepartmentID
	}
	h.submitAuditLog(in)
}
