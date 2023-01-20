package traefik_graphql_limits

import (
  // "fmt"
  "strings"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGraphqlLimitOtherEndpoints(t *testing.T) {
	cfg := CreateConfig()

  ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

  body := `{
    "query": "query { hello }"
  }`

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost",  strings.NewReader(body))

	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

  resp := recorder.Result()

  // fmt.Println(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid status code: %d", resp.StatusCode)
	}
}

// func TestGraphqlLimit(t *testing.T) {
// 	cfg.Headers["X-Host"] = "[[.Host]]"
// 	cfg.Headers["X-Method"] = "[[.Method]]"
// 	cfg.Headers["X-URL"] = "[[.URL]]"
// 	cfg.Headers["X-URL"] = "[[.URL]]"
// 	cfg.Headers["X-Demo"] = "test"
//
// 	ctx := context.Background()
// 	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})
//
// 	handler, err := New(ctx, next, cfg, "demo-plugin")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	recorder := httptest.NewRecorder()
//
// 	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	handler.ServeHTTP(recorder, req)
//
// 	assertHeader(t, req, "X-Host", "localhost")
// 	assertHeader(t, req, "X-URL", "http://localhost")
// 	assertHeader(t, req, "X-Method", "GET")
// 	assertHeader(t, req, "X-Demo", "test")
// }
//
// func assertHeader(t *testing.T, req *http.Request, key, expected string) {
// 	t.Helper()
//
// 	if req.Header.Get(key) != expected {
// 		t.Errorf("invalid header value: %s", req.Header.Get(key))
// 	}
// }
