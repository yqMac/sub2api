// [bmai-fork] audit + organization routes
package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"

	"github.com/gin-gonic/gin"
)

func registerAuditRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if h.Admin.Audit == nil {
		return
	}
	audit := admin.Group("/audit")
	{
		audit.GET("/logs", h.Admin.Audit.List)
		audit.GET("/logs/:id", h.Admin.Audit.Get)

		if h.Admin.AuditSettings != nil {
			audit.GET("/settings", h.Admin.AuditSettings.Get)
			audit.PUT("/settings", h.Admin.AuditSettings.Update)
			audit.GET("/storage", h.Admin.AuditSettings.Storage)
			audit.POST("/cleanup", h.Admin.AuditSettings.Cleanup)
		}
	}
}

func registerOrganizationRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if h.Admin.Organization == nil {
		return
	}
	orgs := admin.Group("/organizations")
	{
		orgs.GET("", h.Admin.Organization.List)
		orgs.POST("", h.Admin.Organization.Create)
		orgs.DELETE("/:id", h.Admin.Organization.Delete)
		orgs.GET("/:id/departments", h.Admin.Organization.DepartmentTree)
		orgs.POST("/:id/departments", h.Admin.Organization.CreateDepartment)
		orgs.PUT("/departments/:id", h.Admin.Organization.UpdateDepartment)
		orgs.DELETE("/departments/:id", h.Admin.Organization.DeleteDepartment)
		orgs.GET("/departments/:id/users", h.Admin.Organization.DepartmentUsers)
		orgs.POST("/departments/:id/users", h.Admin.Organization.AssignUser)
		orgs.DELETE("/departments/:id/users/:userId", h.Admin.Organization.RemoveUser)
	}
}
