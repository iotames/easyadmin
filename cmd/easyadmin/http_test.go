package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kernelapp "github.com/iotames/easyadmin/kernel/app"
	"github.com/iotames/easyadmin/webui"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	runtime, err := kernelapp.New(kernelapp.Options{Name: "easyadmin", Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Stop(context.Background()) })
	return newHandler(runtime, webui.FS())
}

func serve(h http.Handler, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是 JSON: %v (%s)", err, rec.Body.String())
	}
	return body
}

func TestProbes(t *testing.T) {
	h := newTestHandler(t)

	rec := serve(h, http.MethodGet, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz 状态码错误: %d", rec.Code)
	}
	if body := decode(t, rec); body["ok"] != true || body["status"] != "alive" {
		t.Fatalf("healthz 结构不符契约: %v", body)
	}

	rec = serve(h, http.MethodGet, "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("readyz 状态码错误: %d", rec.Code)
	}
	if body := decode(t, rec); body["status"] != "ready" {
		t.Fatalf("readyz 结构不符契约: %v", body)
	}

	notStarted, err := kernelapp.New(kernelapp.Options{Name: "easyadmin"})
	if err != nil {
		t.Fatal(err)
	}
	rec = serve(newHandler(notStarted, webui.FS()), http.MethodGet, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("未启动时 readyz 应为 503，实际 %d", rec.Code)
	}
	if body := decode(t, rec); body["ok"] != false || body["status"] != "not_ready" {
		t.Fatalf("not ready 结构不符契约: %v", body)
	}
}

func TestMetaEndpoint(t *testing.T) {
	h := newTestHandler(t)
	rec := serve(h, http.MethodGet, "/api/meta")
	if rec.Code != http.StatusOK {
		t.Fatalf("meta 状态码错误: %d", rec.Code)
	}
	var body struct {
		OK   bool `json:"ok"`
		Data struct {
			App struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"app"`
			Initialized bool     `json:"initialized"`
			Permissions []string `json:"permissions"`
			Menus       []any    `json:"menus"`
		} `json:"data"`
		Meta struct {
			RequestID string `json:"request_id"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.OK || body.Data.App.Name != "easyadmin" || body.Data.App.Version != "test" {
		t.Fatalf("meta 内容错误: %+v", body)
	}
	if body.Data.Initialized || body.Data.Permissions == nil || body.Data.Menus == nil {
		t.Fatalf("P00 空态字段应为显式空值: %+v", body.Data)
	}
	if body.Meta.RequestID == "" || rec.Header().Get("X-Request-ID") != body.Meta.RequestID {
		t.Fatalf("request_id 未统一: header=%q meta=%q", rec.Header().Get("X-Request-ID"), body.Meta.RequestID)
	}
}

func TestRequestIDPassthrough(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/api/meta", nil)
	req.Header.Set("X-Request-ID", "fixed-id")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "fixed-id" {
		t.Fatalf("应透传请求 ID，实际 %q", got)
	}
	if !strings.Contains(rec.Body.String(), `"request_id":"fixed-id"`) {
		t.Fatalf("meta.request_id 未使用透传值: %s", rec.Body.String())
	}
}

func TestAPIUnknownAndMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	rec := serve(h, http.MethodGet, "/api/unknown")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未知 API 应 404，实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("未知 API 应返回 JSON，实际 %q", ct)
	}
	body := decode(t, rec)
	if body["ok"] != false {
		t.Fatalf("错误响应 ok 应为 false: %v", body)
	}
	if errObj, _ := body["error"].(map[string]any); errObj["code"] != codeNotFound {
		t.Fatalf("错误码应为 %s: %v", codeNotFound, body)
	}

	rec = serve(h, http.MethodPost, "/api/meta")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("错误方法应 405，实际 %d", rec.Code)
	}
	if rec.Header().Get("Allow") == "" {
		t.Fatal("405 应返回 Allow 头")
	}
	if errObj, _ := decode(t, rec)["error"].(map[string]any); errObj["code"] != codeMethodNotAllowed {
		t.Fatalf("405 错误码不符契约: %v", errObj)
	}
}

func TestStaticAssetsAndSPAFallback(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{"/", "/dashboard"} {
		rec := serve(h, http.MethodGet, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s 应回退到首页，实际 %d", target, rec.Code)
		}
		if !strings.Contains(strings.ToLower(rec.Body.String()), "<!doctype html>") {
			t.Fatalf("%s 未返回 HTML 首页", target)
		}
	}

	rec := serve(h, http.MethodGet, "/assets/core/api.js")
	if rec.Code != http.StatusOK {
		t.Fatalf("静态资源应可访问，实际 %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), "export") {
		t.Fatalf("静态资源内容异常: %q", body)
	}

	rec = serve(h, http.MethodGet, "/assets/missing.js")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("缺失静态资源应 404，实际 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("缺失静态资源不应回退 HTML，实际 %q", ct)
	}
}

func TestAPIErrorStatusMapping(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusConflict, http.StatusForbidden} {
		req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
		req = req.WithContext(context.WithValue(req.Context(), requestIDKey{}, "rid"))
		rec := httptest.NewRecorder()
		writeError(rec, req, status, "E_TEST", "测试错误")
		if rec.Code != status {
			t.Fatalf("状态码应为 %d，实际 %d", status, rec.Code)
		}
		var body apiEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.OK || body.Error == nil || body.Error.Code != "E_TEST" || body.Meta.RequestID != "rid" {
			t.Fatalf("错误外壳不符合契约: %+v", body)
		}
	}
}
