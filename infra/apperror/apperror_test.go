package apperror

import (
	"errors"
	"net/http"
	"testing"
)

func TestStatusMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{BadRequest(CodeInvalidArgument, "bad"), http.StatusBadRequest},
		{Validation(map[string][]string{"a": {"x"}}), http.StatusUnprocessableEntity},
		{NotFound(CodeNotFound, "nf"), http.StatusNotFound},
		{Conflict("C", "dup"), http.StatusConflict},
		{Unauthorized("AUTH_REQUIRED", "no"), http.StatusUnauthorized},
		{Forbidden("AUTH_FORBIDDEN", "no"), http.StatusForbidden},
		{RateLimited("AUTH_RATE_LIMITED", "slow"), http.StatusTooManyRequests},
		{Internal(""), http.StatusInternalServerError},
		{errors.New("plain"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		if got := Status(c.err); got != c.want {
			t.Fatalf("%v 状态码 = %d, want %d", c.err, got, c.want)
		}
	}
}

func TestValidationDetails(t *testing.T) {
	e := Validation(map[string][]string{"username": {"不能为空"}})
	fields, ok := e.Details["fields"].(map[string][]string)
	if !ok || len(fields["username"]) != 1 {
		t.Fatalf("Details = %#v", e.Details)
	}
}

func TestWrapCause(t *testing.T) {
	base := errors.New("io boom")
	e := Internal("失败").WithCause(base)
	if !errors.Is(e, base) {
		t.Fatal("Unwrap 未暴露 Cause")
	}
	if _, ok := From(e); !ok {
		t.Fatal("From 未识别 *Error")
	}
}
