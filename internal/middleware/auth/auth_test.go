package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/limes-cloud/manager/api/authorize"
)

func TestStrict(t *testing.T) {
	var tokenHeader string
	handle := func(ctx khttp.Context) error {
		handler := ctx.Middleware(func(context.Context, any) (any, error) {
			return nil, nil
		})
		if _, err := handler(ctx, nil); err != nil {
			return err
		}
		tokenHeader = ctx.Request().Header.Get(authorize.TokenKey)
		return ctx.Result(http.StatusOK, "ok")
	}

	srv := khttp.NewServer(khttp.Middleware(Strict()))
	route := srv.Route("/resource/api")
	route.GET("/store", handle)
	route.OPTIONS("/store", handle)
	route.GET("/file/object-by-key", handle)
	route.GET("/file/read-by-key", handle)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	tests := []struct {
		name       string
		method     string
		path       string
		headers    map[string]string
		wantStatus int
		wantToken  string
	}{
		{name: "store rejects anonymous", method: http.MethodGet, path: "/resource/api/store", wantStatus: http.StatusForbidden},
		{name: "options passes", method: http.MethodOptions, path: "/resource/api/store", wantStatus: http.StatusOK},
		{name: "public redirect", method: http.MethodGet, path: "/resource/api/file/object-by-key", wantStatus: http.StatusOK},
		{name: "public read", method: http.MethodGet, path: "/resource/api/file/read-by-key", wantStatus: http.StatusOK},
		{name: "gateway identity passes", method: http.MethodGet, path: "/resource/api/store", headers: map[string]string{authorize.InfoKey: `{"userId":1}`}, wantStatus: http.StatusOK},
		{name: "bearer token passes", method: http.MethodGet, path: "/resource/api/store", headers: map[string]string{"Authorization": "Bearer token"}, wantStatus: http.StatusOK, wantToken: "token"},
		{name: "raw token passes", method: http.MethodGet, path: "/resource/api/store", headers: map[string]string{"Authorization": "token"}, wantStatus: http.StatusOK, wantToken: "token"},
		{name: "token header passes", method: http.MethodGet, path: "/resource/api/store", headers: map[string]string{authorize.TokenKey: "token"}, wantStatus: http.StatusOK, wantToken: "token"},
		{name: "empty bearer is rejected", method: http.MethodGet, path: "/resource/api/store", headers: map[string]string{"Authorization": "Bearer   "}, wantStatus: http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenHeader = ""
			req, err := http.NewRequest(tt.method, ts.URL+tt.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tokenHeader != tt.wantToken {
				t.Fatalf("token header = %q, want %q", tokenHeader, tt.wantToken)
			}
		})
	}
}

func TestIsPublic(t *testing.T) {
	tests := []struct {
		method string
		path   string
		want   bool
	}{
		{method: http.MethodGet, path: "/resource/api/file/object-by-key", want: true},
		{method: http.MethodGet, path: "/resource/api/file/read-by-key", want: true},
		{method: http.MethodPost, path: "/resource/api/file/object-by-key"},
		{method: http.MethodGet, path: "/resource/api/file/read-by-key/"},
		{method: http.MethodGet, path: "/resource/api/file/url-by-key"},
	}

	for _, tt := range tests {
		if got := isPublic(tt.method, tt.path); got != tt.want {
			t.Errorf("isPublic(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
		}
	}
}

func TestStrictPassesWithoutHTTPTransport(t *testing.T) {
	called := false
	handler := Strict()(func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if _, err := handler(context.Background(), nil); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}
