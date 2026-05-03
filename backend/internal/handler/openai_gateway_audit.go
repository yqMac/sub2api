// [bmai-fork] OpenAI gateway audit helpers
package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// auditAcquireBuffer for OpenAI handler — same shape as gateway_audit.go but on OpenAI handler.
func (h *OpenAIGatewayHandler) auditAcquireBuffer(ctx context.Context) (*service.AuditCaptureBuffer, context.Context) {
	if h.auditLogService == nil || !h.auditLogService.Enabled() {
		return nil, ctx
	}
	buf := service.AcquireAuditCaptureBuffer(h.auditLogService.MaxResponseBytes())
	return buf, service.WithAuditCaptureBuffer(ctx, buf)
}

func (h *OpenAIGatewayHandler) auditReleaseBuffer(buf *service.AuditCaptureBuffer) {
	service.ReleaseAuditCaptureBuffer(buf)
}

// submitAuditFromOpenAIContext bridges OpenAI handler state into AuditLog.
func (h *OpenAIGatewayHandler) submitAuditFromOpenAIContext(
	apiKey *service.APIKey,
	account *service.Account,
	result *service.OpenAIForwardResult,
	auditBuf *service.AuditCaptureBuffer,
	requestBody []byte,
	inboundEndpoint string,
	hasReasoning bool,
) {
	if h.auditLogService == nil || !h.auditLogService.Enabled() || result == nil || apiKey == nil || account == nil {
		return
	}

	var (
		respBody  []byte
		respTotal int
		respTrunc bool
	)
	if auditBuf != nil {
		respBody = auditBuf.Bytes()
		respTotal = auditBuf.Total
		respTrunc = auditBuf.Truncated()
	}

	maxReq := h.auditLogService.MaxRequestBytes()
	reqPreview, reqTrunc := truncatePreview(requestBody, maxReq)

	contentType := service.ClassifyContentType(service.AuditClassifyInput{
		Endpoint:     inboundEndpoint,
		Model:        result.Model,
		HasTools:     detectHasTools(requestBody),
		HasReasoning: hasReasoning,
	})
	if h.auditLogService.ClassifyResponse() && len(respBody) > 0 {
		contentType = service.RefineContentType(contentType, respBody)
	}

	log := &domain.AuditLog{
		RequestID:         result.RequestID,
		UserID:            apiKey.UserID,
		APIKeyID:          apiKey.ID,
		AccountID:         account.ID,
		ContentType:       contentType,
		RequestPreview:    reqPreview,
		RequestBytes:      len(requestBody),
		RequestTruncated:  reqTrunc,
		ResponsePreview:   string(respBody),
		ResponseBytes:     respTotal,
		ResponseTruncated: respTrunc,
		Model:             result.Model,
		Platform:          account.Platform,
		Endpoint:          inboundEndpoint,
		Stream:            result.Stream,
		DurationMs:        int(result.Duration.Milliseconds()),
		InputTokens:       int(result.Usage.InputTokens),
		OutputTokens:      int(result.Usage.OutputTokens),
		StatusCode:        200,
	}
	if apiKey.User != nil {
		log.OrganizationID = apiKey.User.OrganizationID
		log.DepartmentID = apiKey.User.PrimaryDepartmentID
	}
	h.auditLogService.Submit(log)
}
