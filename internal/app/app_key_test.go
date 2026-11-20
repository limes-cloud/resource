package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

func TestCurrentApp(t *testing.T) {
	var (
		gotApp string
		gotErr error
	)
	handle := func(ctx khttp.Context) error {
		gotApp, gotErr = currentApp(ctx)
		return ctx.Result(http.StatusOK, "ok")
	}

	srv := khttp.NewServer()
	route := srv.Route("/resource/api")
	route.GET("/app", handle)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	tests := []struct {
		name     string
		loginApp string
		query    string
		wantApp  string
		wantErr  bool
	}{
		{name: "manager targets another app", loginApp: "manager", query: "?app=resource", wantApp: "resource"},
		{name: "manager defaults to itself", loginApp: "manager", wantApp: "manager"},
		{name: "non-manager cannot override", loginApp: "resource", query: "?app=psou", wantApp: "resource"},
		{name: "missing login app", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotApp, gotErr = "", nil
			req, err := http.NewRequest(http.MethodGet, ts.URL+"/resource/api/app"+tt.query, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.loginApp != "" {
				req.Header.Set("X-App-Key", tt.loginApp)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if (gotErr != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", gotErr, tt.wantErr)
			}
			if gotApp != tt.wantApp {
				t.Fatalf("app = %q, want %q", gotApp, tt.wantApp)
			}
		})
	}
}
