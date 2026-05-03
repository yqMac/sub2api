// [bmai-fork] content type classification at request and response time
package service

import (
	"regexp"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

type AuditClassifyInput struct {
	Endpoint     string
	Model        string
	BillingMode  string
	HasTools     bool
	HasThinking  bool
	HasReasoning bool
}

func ClassifyContentType(in AuditClassifyInput) string {
	switch {
	case in.BillingMode == "image" || isImageEndpoint(in.Endpoint):
		return domain.ContentTypeImage
	case in.HasTools:
		return domain.ContentTypeToolUse
	case in.HasThinking || in.HasReasoning:
		return domain.ContentTypeReasoning
	default:
		return domain.ContentTypeConversation
	}
}

var (
	codeFenceRe = regexp.MustCompile("(?m)```(?:python|go|javascript|typescript|java|cpp|c|rust|bash|sh|sql|json|yaml|html|css|vue|jsx|tsx|ruby|swift|kotlin|scala|php|perl|r|lua|zig|hcl|terraform|dockerfile)")
	scriptRe    = regexp.MustCompile(`(?m)^#!\s*/(?:usr/)?bin/`)
	planRe      = regexp.MustCompile(`(?m)^##\s+\S+`)
)

func RefineContentType(original string, responsePreview []byte) string {
	if len(responsePreview) == 0 {
		return original
	}
	if codeFenceRe.Match(responsePreview) {
		return domain.ContentTypeCode
	}
	if scriptRe.Match(responsePreview) {
		return domain.ContentTypeScript
	}
	if planRe.Match(responsePreview) {
		return domain.ContentTypePlan
	}
	return original
}

func isImageEndpoint(endpoint string) bool {
	return endpoint == "/v1/images/generations" || endpoint == "/v1/images/edits"
}
