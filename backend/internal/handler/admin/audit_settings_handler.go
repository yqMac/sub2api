// [bmai-fork] audit settings handler
package admin

import (
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AuditSettingsHandler struct {
	auditService *service.AuditLogService
}

func NewAuditSettingsHandler(auditService *service.AuditLogService) *AuditSettingsHandler {
	return &AuditSettingsHandler{auditService: auditService}
}

func (h *AuditSettingsHandler) Get(c *gin.Context) {
	cfg := h.auditService.Config()
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": auditSettingsResponse(cfg)})
}

func (h *AuditSettingsHandler) Update(c *gin.Context) {
	var req struct {
		Enabled          *bool `json:"enabled"`
		MaxRequestBytes  *int  `json:"max_request_bytes"`
		MaxResponseBytes *int  `json:"max_response_bytes"`
		CaptureUpstream  *bool `json:"capture_upstream"`
		RetentionDays    *int  `json:"retention_days"`
		ClassifyResponse *bool `json:"classify_response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "invalid request body"})
		return
	}

	cfg := *h.auditService.Config()
	if req.Enabled != nil {
		cfg.Enabled = *req.Enabled
	}
	if req.MaxRequestBytes != nil && *req.MaxRequestBytes >= 1024 {
		cfg.MaxRequestBytes = *req.MaxRequestBytes
	}
	if req.MaxResponseBytes != nil && *req.MaxResponseBytes >= 1024 {
		cfg.MaxResponseBytes = *req.MaxResponseBytes
	}
	if req.CaptureUpstream != nil {
		cfg.CaptureUpstream = *req.CaptureUpstream
	}
	if req.RetentionDays != nil && *req.RetentionDays >= 1 {
		cfg.RetentionDays = *req.RetentionDays
	}
	if req.ClassifyResponse != nil {
		cfg.ClassifyResponse = *req.ClassifyResponse
	}
	h.auditService.UpdateConfig(&cfg)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": auditSettingsResponse(&cfg)})
}

func (h *AuditSettingsHandler) Storage(c *gin.Context) {
	info, err := h.auditService.StorageInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": info})
}

func (h *AuditSettingsHandler) Cleanup(c *gin.Context) {
	var req struct {
		BeforeDays int `json:"before_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.BeforeDays < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "before_days must be >= 1"})
		return
	}
	before := time.Now().AddDate(0, 0, -req.BeforeDays)
	deleted, err := h.auditService.DeleteBefore(c.Request.Context(), before)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"deleted": deleted}})
}

func auditSettingsResponse(cfg *config.AuditConfig) gin.H {
	return gin.H{
		"enabled":            cfg.Enabled,
		"max_request_bytes":  cfg.MaxRequestBytes,
		"max_response_bytes": cfg.MaxResponseBytes,
		"capture_upstream":   cfg.CaptureUpstream,
		"retention_days":     cfg.RetentionDays,
		"classify_response":  cfg.ClassifyResponse,
	}
}
