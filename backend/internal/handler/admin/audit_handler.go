// [bmai-fork] audit log list/detail handler
package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	auditService *service.AuditLogService
}

func NewAuditHandler(auditService *service.AuditLogService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

func (h *AuditHandler) List(c *gin.Context) {
	f := service.AuditLogFilter{
		Page:      intParam(c, "page", 1),
		PageSize:  intParam(c, "page_size", 50),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
		Keyword:   c.Query("keyword"),
		RiskLevel: c.Query("risk_level"),
	}
	if v := c.Query("start_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.StartTime = &t
		}
	}
	if v := c.Query("end_time"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			f.EndTime = &t
		}
	}
	if f.StartTime == nil {
		t := time.Now().Add(-24 * time.Hour)
		f.StartTime = &t
	}
	if v := c.Query("user_id"); v != "" {
		f.UserIDs = parseIntList(v)
	}
	if v := c.Query("organization_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.OrganizationID = &id
		}
	}
	if v := c.Query("department_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.DepartmentID = &id
		}
	}
	f.DepartmentPath = c.Query("department_path")
	if v := c.Query("content_type"); v != "" {
		f.ContentTypes = strings.Split(v, ",")
	}
	if v := c.Query("model"); v != "" {
		f.Models = strings.Split(v, ",")
	}
	if v := c.Query("platform"); v != "" {
		f.Platforms = strings.Split(v, ",")
	}
	if v := c.Query("intercepted"); v == "true" || v == "false" {
		b := v == "true"
		f.Intercepted = &b
	}

	logs, total, err := h.auditService.List(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	items := make([]gin.H, 0, len(logs))
	for _, l := range logs {
		items = append(items, gin.H{
			"id":               l.ID,
			"request_id":       l.RequestID,
			"user_id":          l.UserID,
			"organization_id":  l.OrganizationID,
			"department_id":    l.DepartmentID,
			"department_path":  l.DepartmentPath,
			"content_type":     l.ContentType,
			"model":            l.Model,
			"platform":         l.Platform,
			"endpoint":         l.Endpoint,
			"request_summary":  truncStr(l.RequestPreview, 100),
			"response_summary": truncStr(l.ResponsePreview, 100),
			"request_bytes":    l.RequestBytes,
			"response_bytes":   l.ResponseBytes,
			"input_tokens":     l.InputTokens,
			"output_tokens":    l.OutputTokens,
			"duration_ms":      l.DurationMs,
			"stream":           l.Stream,
			"status_code":      l.StatusCode,
			"risk_level":       l.RiskLevel,
			"intercepted":      l.Intercepted,
			"created_at":       l.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"items":     items,
			"total":     total,
			"page":      f.Page,
			"page_size": f.PageSize,
		},
	})
}

func (h *AuditHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "invalid id"})
		return
	}
	log, err := h.auditService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	if log == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": log})
}

func intParam(c *gin.Context, key string, def int) int {
	v, err := strconv.Atoi(c.Query(key))
	if err != nil || v < 1 {
		return def
	}
	return v
}

func parseIntList(s string) []int64 {
	parts := strings.Split(s, ",")
	var ids []int64
	for _, p := range parts {
		if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
