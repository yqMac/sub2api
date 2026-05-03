// [bmai-fork] audit domain model
package domain

import "time"

// Content type constants
const (
	ContentTypeConversation = "conversation"
	ContentTypeCode         = "code"
	ContentTypeImage        = "image"
	ContentTypeToolUse      = "tool_use"
	ContentTypeReasoning    = "reasoning"
	ContentTypePlan         = "plan"
	ContentTypeScript       = "script"
)

// Risk level constants
const (
	RiskLevelLow    = "low"
	RiskLevelMedium = "medium"
	RiskLevelHigh   = "high"
)

// Audit rule type constants
const (
	AuditRuleTypePreBlock  = "pre_block"
	AuditRuleTypePostAlert = "post_alert"
	AuditRuleTypePostTag   = "post_tag"
)

// Audit action constants
const (
	AuditActionBlock = "block"
	AuditActionWarn  = "warn"
	AuditActionAlert = "alert"
	AuditActionLog   = "log"
	AuditActionTag   = "tag"
)

type AuditLog struct {
	ID             int64
	RequestID      string
	UserID         int64
	APIKeyID       int64
	AccountID      int64
	OrganizationID *int64
	DepartmentID   *int64
	DepartmentPath string
	ContentType    string

	RequestPreview           string
	RequestBytes             int
	RequestTruncated         bool
	ResponsePreview          string
	ResponseBytes            int
	ResponseTruncated        bool
	UpstreamRequestPreview   string
	UpstreamRequestBytes     int
	UpstreamRequestTruncated bool
	UpstreamResponsePreview  string
	UpstreamResponseBytes    int
	UpstreamResponseTruncated bool

	Model        string
	Platform     string
	Endpoint     string
	Stream       bool
	DurationMs   int
	InputTokens  int
	OutputTokens int
	StatusCode   int

	Intercepted     bool
	InterceptReason string
	RiskLevel       string
	RiskTags        []string

	CreatedAt time.Time
}
