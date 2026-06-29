package main_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/router"
)

func closeBody(t *testing.T, resp *http.Response) {
	t.Helper()
	t.Cleanup(func() {
		_ = resp.Body.Close()
	})
}

func TestDemoHTTPRoutes(t *testing.T) {
	srv := httptest.NewServer(router.New())
	t.Cleanup(srv.Close)

	t.Run("health", func(t *testing.T) {
		resp := mustGET(t, srv.URL+"/health")
		closeBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "ok" {
			t.Fatalf("body = %q, want ok", body)
		}
	})

	t.Run("get user by id", func(t *testing.T) {
		resp := mustGET(t, srv.URL+"/api/users/1")
		closeBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var user map[string]any
		decodeJSON(t, resp.Body, &user)
		if user["name"] != "Alice" {
			t.Fatalf("name = %v, want Alice", user["name"])
		}
	})

	t.Run("list users", func(t *testing.T) {
		resp := mustGET(t, srv.URL+"/api/users")
		closeBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var users []map[string]any
		decodeJSON(t, resp.Body, &users)
		if len(users) < 2 {
			t.Fatalf("users count = %d, want >= 2", len(users))
		}
	})

	t.Run("create user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/users", strings.NewReader(`{"name":"Carol","email":"carol@example.com"}`))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		closeBody(t, resp)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want 201", resp.StatusCode)
		}
	})

	t.Run("get item by id", func(t *testing.T) {
		resp := mustGET(t, srv.URL+"/api/items/1")
		closeBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var item map[string]any
		decodeJSON(t, resp.Body, &item)
		if item["name"] != "Widget" {
			t.Fatalf("name = %v, want Widget", item["name"])
		}
	})

	t.Run("list items", func(t *testing.T) {
		resp := mustGET(t, srv.URL+"/api/items")
		closeBody(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var items []map[string]any
		decodeJSON(t, resp.Body, &items)
		if len(items) < 2 {
			t.Fatalf("items count = %d, want >= 2", len(items))
		}
	})
}

func mustGET(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeJSON(t *testing.T, r io.Reader, v any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(v); err != nil {
		t.Fatal(err)
	}
}
