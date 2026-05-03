package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestClassifyContentType(t *testing.T) {
	tests := []struct {
		name string
		in   AuditClassifyInput
		want string
	}{
		{"image endpoint generations", AuditClassifyInput{Endpoint: "/v1/images/generations"}, domain.ContentTypeImage},
		{"image endpoint edits", AuditClassifyInput{Endpoint: "/v1/images/edits"}, domain.ContentTypeImage},
		{"image billing mode", AuditClassifyInput{BillingMode: "image"}, domain.ContentTypeImage},
		{"tools present", AuditClassifyInput{HasTools: true}, domain.ContentTypeToolUse},
		{"thinking enabled", AuditClassifyInput{HasThinking: true}, domain.ContentTypeReasoning},
		{"reasoning enabled", AuditClassifyInput{HasReasoning: true}, domain.ContentTypeReasoning},
		{"default conversation", AuditClassifyInput{}, domain.ContentTypeConversation},
		{"tools + image prefers image", AuditClassifyInput{HasTools: true, BillingMode: "image"}, domain.ContentTypeImage},
		{"tools + thinking prefers tools", AuditClassifyInput{HasTools: true, HasThinking: true}, domain.ContentTypeToolUse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ClassifyContentType(tt.in))
		})
	}
}

func TestRefineContentType(t *testing.T) {
	tests := []struct {
		name     string
		original string
		preview  string
		want     string
	}{
		{"empty preview keeps original", domain.ContentTypeConversation, "", domain.ContentTypeConversation},
		{"python code fence", domain.ContentTypeConversation, "Here is the code:\n```python\nprint('hi')\n```", domain.ContentTypeCode},
		{"go code fence", domain.ContentTypeConversation, "```go\nfunc main() {}\n```", domain.ContentTypeCode},
		{"bash code fence", domain.ContentTypeConversation, "```bash\necho hi\n```", domain.ContentTypeCode},
		{"shebang script", domain.ContentTypeConversation, "#!/bin/bash\nset -e\necho hi", domain.ContentTypeScript},
		{"usr bin shebang", domain.ContentTypeConversation, "#!/usr/bin/env python3\nimport sys", domain.ContentTypeScript},
		{"markdown plan", domain.ContentTypeConversation, "## Step 1\n- item\n## Step 2\n- item", domain.ContentTypePlan},
		{"no match keeps original", domain.ContentTypeToolUse, "just some text", domain.ContentTypeToolUse},
		{"code fence takes priority over plan", domain.ContentTypeConversation, "## Overview\n```python\nprint(1)\n```", domain.ContentTypeCode},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RefineContentType(tt.original, []byte(tt.preview)))
		})
	}
}
