// [bmai-fork] organization & department handler (manual CRUD for P0)
package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type OrganizationHandler struct {
	orgService  *service.OrganizationService
	syncService *service.FeishuOrgSyncService
}

func NewOrganizationHandler(orgService *service.OrganizationService, syncService *service.FeishuOrgSyncService) *OrganizationHandler {
	return &OrganizationHandler{orgService: orgService, syncService: syncService}
}

// GET /api/admin/organizations
func (h *OrganizationHandler) List(c *gin.Context) {
	orgs, err := h.orgService.ListOrganizations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": orgs})
}

// POST /api/admin/organizations
func (h *OrganizationHandler) Create(c *gin.Context) {
	var req struct {
		TenantKey string `json:"tenant_key" binding:"required"`
		Name      string `json:"name" binding:"required"`
		Type      string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	if req.Type == "" {
		req.Type = "manual"
	}
	org := &domain.Organization{TenantKey: req.TenantKey, Name: req.Name, Type: req.Type}
	result, err := h.orgService.UpsertOrganization(c.Request.Context(), org)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// DELETE /api/admin/organizations/:id
func (h *OrganizationHandler) Delete(c *gin.Context) {
	id := mustInt64(c.Param("id"))
	if err := h.orgService.DeleteOrganization(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

// GET /api/admin/organizations/:id/departments
func (h *OrganizationHandler) DepartmentTree(c *gin.Context) {
	orgID := mustInt64(c.Param("id"))
	depts, err := h.orgService.ListDepartmentTree(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": depts})
}

// POST /api/admin/organizations/:id/departments
func (h *OrganizationHandler) CreateDepartment(c *gin.Context) {
	orgID := mustInt64(c.Param("id"))
	var req struct {
		ParentID   *int64 `json:"parent_id"`
		ExternalID string `json:"external_id" binding:"required"`
		Name       string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	dept := &domain.Department{
		OrganizationID: orgID,
		ParentID:       req.ParentID,
		ExternalID:     req.ExternalID,
		Name:           req.Name,
	}
	result, err := h.orgService.UpsertDepartmentWithPath(c.Request.Context(), dept)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// PUT /api/admin/organizations/departments/:id
func (h *OrganizationHandler) UpdateDepartment(c *gin.Context) {
	deptID := mustInt64(c.Param("id"))
	var req struct {
		Name     string `json:"name"`
		ParentID *int64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	dept, err := h.orgService.GetDepartmentByID(c.Request.Context(), deptID)
	if err != nil || dept == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "department not found"})
		return
	}
	if req.Name != "" {
		dept.Name = req.Name
	}
	dept.ParentID = req.ParentID
	result, err := h.orgService.UpsertDepartmentWithPath(c.Request.Context(), dept)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// DELETE /api/admin/organizations/departments/:id
func (h *OrganizationHandler) DeleteDepartment(c *gin.Context) {
	deptID := mustInt64(c.Param("id"))
	if err := h.orgService.DeleteDepartment(c.Request.Context(), deptID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

// GET /api/admin/organizations/departments/:id/users
func (h *OrganizationHandler) DepartmentUsers(c *gin.Context) {
	deptID := mustInt64(c.Param("id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	users, total, err := h.orgService.GetDepartmentUsers(c.Request.Context(), deptID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"items": users, "total": total, "page": page, "page_size": pageSize},
	})
}

// POST /api/admin/organizations/departments/:id/users
func (h *OrganizationHandler) AssignUser(c *gin.Context) {
	deptID := mustInt64(c.Param("id"))
	var req struct {
		UserID     int64  `json:"user_id" binding:"required"`
		IsPrimary  bool   `json:"is_primary"`
		Role       string `json:"role"`
		EmployeeID string `json:"employee_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	ud := &domain.UserDepartment{
		UserID:       req.UserID,
		DepartmentID: deptID,
		IsPrimary:    req.IsPrimary,
		Role:         req.Role,
		EmployeeID:   req.EmployeeID,
	}
	if err := h.orgService.AssignUserToDepartment(c.Request.Context(), ud); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

// DELETE /api/admin/organizations/departments/:id/users/:userId
func (h *OrganizationHandler) RemoveUser(c *gin.Context) {
	deptID := mustInt64(c.Param("id"))
	userID := mustInt64(c.Param("userId"))
	if err := h.orgService.RemoveUserFromDepartment(c.Request.Context(), userID, deptID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

func mustInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// POST /api/admin/organizations/:id/feishu/config — save Feishu credentials
func (h *OrganizationHandler) FeishuConfig(c *gin.Context) {
	orgID := mustInt64(c.Param("id"))
	var req struct {
		AppID     string `json:"app_id" binding:"required"`
		AppSecret string `json:"app_secret" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	org, err := h.orgService.GetByID(c.Request.Context(), orgID)
	if err != nil || org == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "organization not found"})
		return
	}
	if org.Metadata == nil {
		org.Metadata = make(map[string]any)
	}
	org.Metadata["feishu_app_id"] = req.AppID
	org.Metadata["feishu_app_secret"] = req.AppSecret
	if _, err := h.orgService.UpsertOrganization(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

// POST /api/admin/organizations/:id/feishu/test — test Feishu connection
func (h *OrganizationHandler) FeishuTest(c *gin.Context) {
	orgID := mustInt64(c.Param("id"))
	org, err := h.orgService.GetByID(c.Request.Context(), orgID)
	if err != nil || org == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "organization not found"})
		return
	}
	appID, _ := org.Metadata["feishu_app_id"].(string)
	appSecret, _ := org.Metadata["feishu_app_secret"].(string)
	if appID == "" || appSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "feishu credentials not configured"})
		return
	}
	count, err := h.syncService.TestConnection(c.Request.Context(), appID, appSecret)
	if err != nil {
		hint := err.Error()
		if strings.Contains(hint, "40004") || strings.Contains(hint, "no dept authority") {
			hint = "飞书应用缺少通讯录权限。请在飞书开放平台 → 应用 → 权限管理中开通：contact:department.base:readonly 和 contact:user.base:readonly，并设置通讯录授权范围为「全部员工」"
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": false, "error": hint}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"success": true, "department_count": count}})
}

// POST /api/admin/organizations/:id/feishu/sync — trigger full sync
func (h *OrganizationHandler) FeishuSync(c *gin.Context) {
	orgID := mustInt64(c.Param("id"))
	org, err := h.orgService.GetByID(c.Request.Context(), orgID)
	if err != nil || org == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "msg": "organization not found"})
		return
	}
	appID, _ := org.Metadata["feishu_app_id"].(string)
	appSecret, _ := org.Metadata["feishu_app_secret"].(string)
	if appID == "" || appSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "msg": "feishu credentials not configured"})
		return
	}
	result, err := h.syncService.SyncOrganization(c.Request.Context(), org, appID, appSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GET /api/admin/organizations/:id/feishu/status — get sync status
func (h *OrganizationHandler) FeishuStatus(c *gin.Context) {
	orgID := mustInt64(c.Param("id"))
	status, err := h.syncService.GetSyncStatus(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": status})
}
