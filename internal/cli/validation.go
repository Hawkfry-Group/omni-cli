package cli

import (
	"context"
	"net/http"
	"strconv"

	"github.com/omni-co/omni-cli/internal/client"
)

type capabilityCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	HTTPStatus int    `json:"http_status,omitempty"`
	Message    string `json:"message,omitempty"`
}

type validationSummary struct {
	Base         capabilityCheck  `json:"base"`
	Query        capabilityCheck  `json:"query"`
	Admin        capabilityCheck  `json:"admin"`
	RateLimit    map[string]int64 `json:"rate_limit,omitempty"`
	Capabilities []string         `json:"capabilities"`
}

type validationFailure struct {
	Code    string
	Message string
	Details any
}

func (e *validationFailure) Error() string { return e.Message }

func collectValidation(ctx context.Context, api *client.Client, tokenType string, includeAdmin bool) (validationSummary, *validationFailure) {
	summary := validationSummary{
		Capabilities: make([]string, 0, 3),
	}

	baseResp, err := api.BaseProbe(ctx)
	if err != nil {
		return summary, &validationFailure{Code: codeNetworkError, Message: "base probe request failed", Details: map[string]any{"error": err.Error()}}
	}
	summary.Base.HTTPStatus = baseResp.StatusCode()
	summary.Base.Name = "base_api"
	summary.RateLimit = parseRateLimit(baseResp.HTTPResponse)
	if baseResp.StatusCode() == http.StatusUnauthorized {
		summary.Base.Status = "fail"
		summary.Base.Message = "unauthorized"
		return summary, &validationFailure{Code: codeAuthUnauthorized, Message: "token unauthorized for Omni API", Details: client.ParseBody(baseResp.Body)}
	}
	if baseResp.StatusCode() >= 200 && baseResp.StatusCode() < 300 {
		summary.Base.Status = "pass"
		summary.Base.Message = "authenticated"
		summary.Capabilities = append(summary.Capabilities, "base_api")
	} else if baseResp.StatusCode() == http.StatusForbidden {
		summary.Base.Status = "warn"
		summary.Base.Message = "authenticated but access to content listing denied"
		summary.Capabilities = append(summary.Capabilities, "base_api")
	} else {
		summary.Base.Status = "fail"
		summary.Base.Message = "unexpected response"
	}

	queryResp, err := api.QueryProbe(ctx)
	if err != nil {
		return summary, &validationFailure{Code: codeNetworkError, Message: "query probe request failed", Details: map[string]any{"error": err.Error()}}
	}
	summary.Query.Name = "query_api"
	summary.Query.HTTPStatus = queryResp.StatusCode()
	switch queryResp.StatusCode() {
	case http.StatusUnauthorized:
		summary.Query.Status = "fail"
		summary.Query.Message = "unauthorized"
		return summary, &validationFailure{Code: codeAuthUnauthorized, Message: "token unauthorized for query API", Details: client.ParseBody(queryResp.Body)}
	case http.StatusForbidden:
		summary.Query.Status = "fail"
		summary.Query.Message = "permission denied"
	case http.StatusBadRequest:
		summary.Query.Status = "pass"
		summary.Query.Message = "endpoint reachable and authenticated"
		summary.Capabilities = append(summary.Capabilities, "query_api")
	default:
		if queryResp.StatusCode() >= 200 && queryResp.StatusCode() < 300 {
			summary.Query.Status = "pass"
			summary.Query.Message = "query API available"
			summary.Capabilities = append(summary.Capabilities, "query_api")
		} else {
			summary.Query.Status = "warn"
			summary.Query.Message = "unexpected query probe response"
		}
	}

	summary.Admin.Name = "admin_scim"
	if !includeAdmin {
		summary.Admin.Status = "skipped"
		summary.Admin.Message = "admin check skipped for PAT tokens"
		return summary, nil
	}

	adminResp, err := api.AdminProbe(ctx)
	if err != nil {
		return summary, &validationFailure{Code: codeNetworkError, Message: "admin probe request failed", Details: map[string]any{"error": err.Error()}}
	}
	summary.Admin.HTTPStatus = adminResp.StatusCode()
	switch adminResp.StatusCode() {
	case http.StatusUnauthorized:
		summary.Admin.Status = "fail"
		summary.Admin.Message = "unauthorized"
		return summary, &validationFailure{Code: codeAuthUnauthorized, Message: "token unauthorized for admin SCIM API", Details: client.ParseBody(adminResp.Body)}
	case http.StatusForbidden:
		summary.Admin.Status = "fail"
		summary.Admin.Message = "admin permission denied"
	case http.StatusNotFound:
		summary.Admin.Status = "warn"
		summary.Admin.Message = "SCIM endpoint unavailable on this instance"
	default:
		if adminResp.StatusCode() >= 200 && adminResp.StatusCode() < 300 {
			summary.Admin.Status = "pass"
			summary.Admin.Message = "admin SCIM API available"
			summary.Capabilities = append(summary.Capabilities, "admin_scim")
		} else {
			summary.Admin.Status = "warn"
			summary.Admin.Message = "unexpected admin probe response"
		}
	}

	_ = tokenType
	return summary, nil
}

func parseRateLimit(resp *http.Response) map[string]int64 {
	if resp == nil {
		return nil
	}
	keys := []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"}
	out := map[string]int64{}
	for _, k := range keys {
		v := resp.Header.Get(k)
		if v == "" {
			continue
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue
		}
		switch k {
		case "X-RateLimit-Limit":
			out["limit"] = n
		case "X-RateLimit-Remaining":
			out["remaining"] = n
		case "X-RateLimit-Reset":
			out["reset"] = n
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
