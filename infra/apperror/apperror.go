// Package apperror 是业务错误的唯一入口：稳定 Code、用户可读 Message、结构化 Details 与 HTTP 状态映射。
// 内部 Cause 只用于日志，不直接回显给用户。
package apperror

import (
	"errors"
	"net/http"
)

// 通用错误码常量；业务模块在各自命名空间内定义专属 Code。
const (
	CodeInvalidArgument  = "COMMON_INVALID_ARGUMENT"
	CodeValidationFailed = "COMMON_VALIDATION_FAILED"
	CodeNotFound         = "COMMON_NOT_FOUND"
	CodeMethodNotAllowed = "COMMON_METHOD_NOT_ALLOWED"
	CodeInternal         = "COMMON_INTERNAL"
	CodeUnavailable      = "COMMON_UNAVAILABLE"
)

// Error 是带 HTTP 语义的业务错误。Message 面向用户，Cause 只进日志。
type Error struct {
	Code       string
	Message    string
	Details    map[string]any
	HTTPStatus int
	cause      error
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.Code + ": " + e.Message + ": " + e.cause.Error()
	}
	return e.Code + ": " + e.Message
}

// Unwrap 暴露内部 Cause。
func (e *Error) Unwrap() error { return e.cause }

// WithCause 附加内部原因（仅日志可见）。
func (e *Error) WithCause(err error) *Error {
	clone := *e
	clone.cause = err
	return &clone
}

// WithDetails 附加结构化详情（会返回副本，避免共享可变 map）。
func (e *Error) WithDetails(details map[string]any) *Error {
	clone := *e
	clone.Details = details
	return &clone
}

// New 构造自定义错误。
func New(status int, code, message string) *Error {
	return &Error{Code: code, Message: message, HTTPStatus: status}
}

// BadRequest 400：请求格式/结构/类型错误。
func BadRequest(code, message string) *Error {
	return New(http.StatusBadRequest, code, message)
}

// Validation 422：绑定成功后的字段或跨字段校验失败。
func Validation(fields map[string][]string) *Error {
	return &Error{
		Code:       CodeValidationFailed,
		Message:    "请求字段校验失败",
		HTTPStatus: http.StatusUnprocessableEntity,
		Details:    map[string]any{"fields": fields},
	}
}

// NotFound 404。
func NotFound(code, message string) *Error {
	return New(http.StatusNotFound, code, message)
}

// Conflict 409：已存在数据或资源状态冲突。
func Conflict(code, message string) *Error {
	return New(http.StatusConflict, code, message)
}

// Unauthorized 401。
func Unauthorized(code, message string) *Error {
	return New(http.StatusUnauthorized, code, message)
}

// Forbidden 403。
func Forbidden(code, message string) *Error {
	return New(http.StatusForbidden, code, message)
}

// RateLimited 429。
func RateLimited(code, message string) *Error {
	return New(http.StatusTooManyRequests, code, message)
}

// Internal 500。
func Internal(message string) *Error {
	if message == "" {
		message = "服务器内部错误"
	}
	return New(http.StatusInternalServerError, CodeInternal, message)
}

// From 判断 err 是否为 *Error。
func From(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// Status 返回 err 对应的 HTTP 状态码，未知错误一律 500。
func Status(err error) int {
	if e, ok := From(err); ok && e.HTTPStatus != 0 {
		return e.HTTPStatus
	}
	return http.StatusInternalServerError
}
