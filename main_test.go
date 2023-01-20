package traefik_graphql_limits

import (
	// "fmt"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGraphqlLimitGetEndpoint(t *testing.T) {
	cfg := CreateConfig()

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid status code: %d", resp.StatusCode)
	}
}

func TestGraphqlLimitOtherPath(t *testing.T) {
	cfg := CreateConfig()
	cfg.GraphQLPath = "/graphql"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/api/v1", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid status code: %d", resp.StatusCode)
	}
}

func TestGraphqlLimitDepthNotSet(t *testing.T) {
	cfg := CreateConfig()
	cfg.GraphQLPath = "/graphql"
	cfg.DepthLimit = 0

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/graphql", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid status code: %d", resp.StatusCode)
	}
}

func TestGraphqlLimitDepthSet(t *testing.T) {
	cfg := CreateConfig()
	cfg.GraphQLPath = "/graphql"
	cfg.DepthLimit = 5

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/graphql", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid status code: %d", resp.StatusCode)
	}
}
