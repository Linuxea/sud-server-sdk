package sud

import (
	"fmt"
	"net/http"
)

// APIError 出站请求失败的统一错误类型，业务方可用 errors.As 细分处理。
type APIError struct {
	// URL 请求地址
	URL string
	// HTTPStatus HTTP 状态码（网络失败时为 0）
	HTTPStatus int
	// RetCode Sud 业务错误码（HTTP 层失败时为 0）
	RetCode int
	// RetMsg 错误信息
	RetMsg string
	// Body 原始响应体（截断保留，便于排查）
	Body string
}

func (e *APIError) Error() string {
	switch {
	case e.HTTPStatus != 0 && e.HTTPStatus != http.StatusOK:
		return fmt.Sprintf("sud: %s http status %d: %s", e.URL, e.HTTPStatus, e.Body)
	case e.RetCode != 0:
		return fmt.Sprintf("sud: %s ret_code=%d ret_msg=%s", e.URL, e.RetCode, e.RetMsg)
	default:
		return fmt.Sprintf("sud: %s: %s", e.URL, e.RetMsg)
	}
}
