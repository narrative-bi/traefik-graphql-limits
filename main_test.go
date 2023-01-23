package traefik_graphql_limits

import (
	// "fmt"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	// "log"
	"io"
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

func TestGraphqlLimitInvalidQuery(t *testing.T) {
	cfg := CreateConfig()
	cfg.GraphQLPath = "/graphql"
	cfg.DepthLimit = 5

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

	body := `{
    "query""&: query { __schema { queryType { name } } }"
  }`

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/graphql", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("invalid status code: %d", resp.StatusCode)
	}
}

func TestGraphqlLimitDepthLimitNotReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.GraphQLPath = "/graphql"
	cfg.DepthLimit = 3

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

	body := `
    query GetUser($id: ID!, $page: Pagination) {
      user(id: $id) {
        name
        email
        phone
        address
        friend(page: $page) {
          id
          name
          email
        }
      }
    }
  `
	// body := `
	//   query GetUser($id: ID!, $page: Pagination) {
	//     user(id: $id) {
	//       name
	//       friend(page: $page) {
	//         id
	//       }
	//     }
	//   }
	// `
	// body := `
	//   query namedQuery($foo: ComplexFooType, $bar: Bar = DefaultBarValue) {
	//     customUser: user(id: [987, 654]) {
	//       id,
	//       ... on User @defer {
	//         field2 {
	//           id ,
	//           alias: field1(first:10, after:$foo,) @include(if: $foo) {
	//             id,
	//             ...frag
	//           }
	//         }
	//       }
	//       ... @skip(unless: $foo) {
	//         id
	//       }
	//       ... {
	//         id
	//       }
	//     }
	//   }
	// `
	// body := `
	//   mutation favPost {
	//     fav(post: 123) @defer {
	//       post {
	//         id
	//       }
	//     }
	//   }
	// `
	// body := `
	//   subscription PostFavSubscription($input: StoryLikeSubscribeInput) {
	//     postFavSubscribe(input: $input) {
	//       post {
	//         favers {
	//           count
	//         }
	//         favSentence {
	//           text
	//         }
	//       }
	//     }
	//   }
	// `
	// body := `
	//   fragment frag on Follower {
	//     foo(size: $size, bar: $b, obj: {key: "value"})
	//   }
	// `

	// body := `
	//   {
	//     unnamed(truthyVal: true, falseyVal: false),
	//     query
	//   }
	// `

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost/graphql", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	resp := recorder.Result()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("invalid response (code: %d, body: %s)", resp.StatusCode, respBody)
	}
}
