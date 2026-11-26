// Package httpapi 是 HTTP 路由、中间件、响应外壳与请求绑定的唯一入口。
package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/iotames/easyadmin/infra/apperror"
)

// Meta 是统一响应元信息；request_id 只出现在这里。
type Meta struct {
	RequestID string `json:"request_id"`
	Warning   string `json:"warning,omitempty"`
}

// ErrorBody 是失败响应体。
type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Envelope 是 /api/ 成功与失败响应的统一外壳。
type Envelope struct {
	OK    bool       `json:"ok"`
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
	Meta  Meta       `json:"meta"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// WriteOK 输出成功响应。
func WriteOK(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, http.StatusOK, Envelope{OK: true, Data: data, Meta: Meta{RequestID: RequestIDFrom(r)}})
}

// WriteError 输出失败响应。
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, Envelope{Error: &ErrorBody{Code: code, Message: message}, Meta: Meta{RequestID: RequestIDFrom(r)}})
}

// WriteAppError 按 apperror 的状态码与详情输出失败响应；未知错误按 500 处理且不回显内部原因。
func WriteAppError(w http.ResponseWriter, r *http.Request, err error) {
	if e, ok := apperror.From(err); ok {
		writeJSON(w, apperror.Status(err), Envelope{
			Error: &ErrorBody{Code: e.Code, Message: e.Message, Details: e.Details},
			Meta:  Meta{RequestID: RequestIDFrom(r)},
		})
		return
	}
	WriteError(w, r, http.StatusInternalServerError, apperror.CodeInternal, "服务器内部错误")
}

// WriteProbe 输出探针响应；探针非 /api/ 端点，使用独立结构。
func WriteProbe(w http.ResponseWriter, status int, body any) { writeJSON(w, status, body) }
