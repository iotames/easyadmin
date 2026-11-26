package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/iotames/easyadmin/infra/apperror"
)

// maxJSONBody 限制 JSON 请求体大小，防止超大输入。
const maxJSONBody = 1 << 20

// DecodeJSON 绑定 JSON 请求体：格式错误、未知字段、类型转换失败返回 400。
// 不在本函数内做字段/领域校验；绑定成功后的校验失败由 Validate 返回 422。
func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return apperror.BadRequest(apperror.CodeInvalidArgument, "请求体不能为空")
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return apperror.BadRequest(apperror.CodeInvalidArgument, "请求格式错误："+err.Error())
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return apperror.BadRequest(apperror.CodeInvalidArgument, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

// Validate 在字段错误非空时返回 422；字段路径为 API 对外字段名。
func Validate(fields map[string][]string) error {
	if len(fields) == 0 {
		return nil
	}
	return apperror.Validation(fields)
}

// Query 返回去空白的查询参数。
func Query(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}
