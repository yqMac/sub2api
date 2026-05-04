// [bmai-fork] Feishu Contact API client for department/user sync
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

type FeishuContactClient struct {
	appID     string
	appSecret string
	baseURL   string
}

func NewFeishuContactClient(appID, appSecret string) *FeishuContactClient {
	return &FeishuContactClient{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   "https://open.feishu.cn",
	}
}

type FeishuDepartment struct {
	DepartmentID       string
	Name               string
	ParentDepartmentID string
	MemberCount        int
	Status             string
}

type FeishuUser struct {
	UserID          string
	UnionID         string
	Name            string
	Email           string
	EnterpriseEmail string
	Mobile          string
	DepartmentIDs   []string
	EmployeeNo      string
	JobTitle        string
	Avatar          string
}

func (c *FeishuContactClient) GetTenantAccessToken(ctx context.Context) (string, error) {
	client := req.C().SetTimeout(15 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetBodyJsonMarshal(map[string]string{
			"app_id":     c.appID,
			"app_secret": c.appSecret,
		}).
		Post(c.baseURL + "/open-apis/auth/v3/tenant_access_token/internal")
	if err != nil {
		return "", fmt.Errorf("feishu token request failed: %w", err)
	}
	body := resp.String()
	code := gjson.Get(body, "code").Int()
	if code != 0 {
		msg := gjson.Get(body, "msg").String()
		return "", fmt.Errorf("feishu token error: code=%d msg=%s", code, msg)
	}
	token := gjson.Get(body, "tenant_access_token").String()
	if token == "" {
		return "", fmt.Errorf("feishu token empty in response")
	}
	return token, nil
}

func (c *FeishuContactClient) ListDepartments(ctx context.Context, token string) ([]FeishuDepartment, error) {
	var all []FeishuDepartment
	pageToken := ""
	client := req.C().SetTimeout(30 * time.Second)

	for {
		r := client.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+token).
			SetQueryParam("department_id_type", "open_department_id").
			SetQueryParam("parent_department_id", "0").
			SetQueryParam("fetch_child", "true").
			SetQueryParam("page_size", "50")
		if pageToken != "" {
			r.SetQueryParam("page_token", pageToken)
		}

		resp, err := r.Get(c.baseURL + "/open-apis/contact/v3/departments")
		if err != nil {
			return nil, fmt.Errorf("feishu list departments failed: %w", err)
		}
		body := resp.String()
		code := gjson.Get(body, "code").Int()
		if code != 0 {
			return nil, fmt.Errorf("feishu departments error: code=%d msg=%s", code, gjson.Get(body, "msg").String())
		}

		items := gjson.Get(body, "data.items")
		items.ForEach(func(_, item gjson.Result) bool {
			all = append(all, FeishuDepartment{
				DepartmentID:       item.Get("open_department_id").String(),
				Name:               item.Get("name").String(),
				ParentDepartmentID: item.Get("parent_department_id").String(),
				MemberCount:        int(item.Get("member_count").Int()),
			})
			return true
		})

		if !gjson.Get(body, "data.has_more").Bool() {
			break
		}
		pageToken = gjson.Get(body, "data.page_token").String()
		if pageToken == "" {
			break
		}
	}
	return all, nil
}

func (c *FeishuContactClient) ListUsersByDepartment(ctx context.Context, token, departmentID string) ([]FeishuUser, error) {
	var all []FeishuUser
	pageToken := ""
	client := req.C().SetTimeout(30 * time.Second)

	for {
		r := client.R().
			SetContext(ctx).
			SetHeader("Authorization", "Bearer "+token).
			SetQueryParam("department_id_type", "open_department_id").
			SetQueryParam("department_id", departmentID).
			SetQueryParam("user_id_type", "open_id").
			SetQueryParam("page_size", "50")
		if pageToken != "" {
			r.SetQueryParam("page_token", pageToken)
		}

		resp, err := r.Get(c.baseURL + "/open-apis/contact/v3/users/find_by_department")
		if err != nil {
			return nil, fmt.Errorf("feishu list users failed: %w", err)
		}
		body := resp.String()
		code := gjson.Get(body, "code").Int()
		if code != 0 {
			return nil, fmt.Errorf("feishu users error: code=%d msg=%s", code, gjson.Get(body, "msg").String())
		}

		items := gjson.Get(body, "data.items")
		items.ForEach(func(_, item gjson.Result) bool {
			var deptIDs []string
			item.Get("department_ids").ForEach(func(_, d gjson.Result) bool {
				deptIDs = append(deptIDs, d.String())
				return true
			})
			all = append(all, FeishuUser{
				UserID:          item.Get("open_id").String(),
				UnionID:         item.Get("union_id").String(),
				Name:            item.Get("name").String(),
				Email:           item.Get("email").String(),
				EnterpriseEmail: item.Get("enterprise_email").String(),
				Mobile:          item.Get("mobile").String(),
				DepartmentIDs:   deptIDs,
				EmployeeNo:      item.Get("employee_no").String(),
				JobTitle:        item.Get("job_title").String(),
				Avatar:          item.Get("avatar.avatar_origin").String(),
			})
			return true
		})

		if !gjson.Get(body, "data.has_more").Bool() {
			break
		}
		pageToken = gjson.Get(body, "data.page_token").String()
		if pageToken == "" {
			break
		}
	}
	return all, nil
}
