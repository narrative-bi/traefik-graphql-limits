// Package traefikgraphqllimits provides a Traefik plugin to limit the depth of a GraphQL query
package traefikgraphqllimits

import (
	"context"
	"io"
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
		t.Errorf("invalid  code: %d", resp.StatusCode)
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
		t.Errorf("invalid  code: %d", resp.StatusCode)
	}
}

func RunGraphqlLimitsTest(t *testing.T, cfg *Config, body string, expectedCode int) {
	t.Helper()

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := New(ctx, next, cfg, "traefik-graphql-limits-plugin")
	if err != nil {
		t.Fatal(err)
	}

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

	if resp.StatusCode != expectedCode {
		t.Errorf("invalid response (code: %d, body: %s)", resp.StatusCode, respBody)
	}
}

func TestGraphqlLimitDepthNotSet(t *testing.T) {
	cfg := CreateConfig()
	cfg.DepthLimit = 0

	body := ""

	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}

// func TestGraphqlLimitInvalidQuery(t *testing.T) {
// 	cfg := CreateConfig()
// 	cfg.DepthLimit = 5
//
// 	body := `{
//     "query""&: query { __schema { queryType { name } } }"
//   }`
//
// 	RunGraphqlLimitsTest(t, cfg, body, http.StatusBadRequest)
// }

func TestGraphqlLimitDepthLimitNotReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.DepthLimit = 3

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

	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}

func TestGraphqlLimitDepthLimitReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.DepthLimit = 1

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

	RunGraphqlLimitsTest(t, cfg, body, http.StatusBadRequest)
}

func TestGraphqlLimitDepthLimitEqual(t *testing.T) {
	cfg := CreateConfig()
	cfg.DepthLimit = 2

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

	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}

func TestGraphqlBatchLimitReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.BatchLimit = 1

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

	  query namedQuery($foo: ComplexFooType, $bar: Bar = DefaultBarValue) {
	    customUser: user(id: [987, 654]) {
	      id,
	      ... on User @defer {
	        field2 {
	          id ,
	          alias: field1(first:10, after:$foo,) @include(if: $foo) {
	            id,
	            ...frag
	          }
	        }
	      }
	      ... @skip(unless: $foo) {
	        id
	      }
	      ... {
	        id
	      }
	    }
	  }

	  fragment frag on Follower {
	    foo(size: $size, bar: $b, obj: {key: "value"})
	  }

	  {
	    unnamed(truthyVal: true, falseyVal: false),
	    query
	  }

  `

	RunGraphqlLimitsTest(t, cfg, body, http.StatusBadRequest)
}

func TestGraphqlBatchLimitNotReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.BatchLimit = 3

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

	  query namedQuery($foo: ComplexFooType, $bar: Bar = DefaultBarValue) {
	    customUser: user(id: [987, 654]) {
	      id,
	      ... on User @defer {
	        field2 {
	          id ,
	          alias: field1(first:10, after:$foo,) @include(if: $foo) {
	            id,
	            ...frag
	          }
	        }
	      }
	      ... @skip(unless: $foo) {
	        id
	      }
	      ... {
	        id
	      }
	    }
	  }
  `

	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}

func TestGraphqlBatchLimitEqual(t *testing.T) {
	cfg := CreateConfig()
	cfg.BatchLimit = 2

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

	  query namedQuery($foo: ComplexFooType, $bar: Bar = DefaultBarValue) {
	    customUser: user(id: [987, 654]) {
	      id,
	      ... on User @defer {
	        field2 {
	          id ,
	          alias: field1(first:10, after:$foo,) @include(if: $foo) {
	            id,
	            ...frag
	          }
	        }
	      }
	      ... @skip(unless: $foo) {
	        id
	      }
	      ... {
	        id
	      }
	    }
	  }
  `

	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}

func TestGraphqlNodeLimitReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.NodeLimit = 2

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
        posts {
          id
        }
      }
    }

	  {
	    unnamed(truthyVal: true, falseyVal: false),
	    query
	  }
  `
	RunGraphqlLimitsTest(t, cfg, body, http.StatusBadRequest)
}

func TestGraphqlNodeLimitNotReached(t *testing.T) {
	cfg := CreateConfig()
	cfg.NodeLimit = 10

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
        posts {
          id
        }
      }
    }

    subscription PostFavSubscription($input: StoryLikeSubscribeInput) {
      postFavSubscribe(input: $input) {
        post {
          favers {
            count
          }
          favSentence {
            text
          }
        }
      }
    }

	  {
	    unnamed(truthyVal: true, falseyVal: false),
	    query
	  }
  `
	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}

func TestGraphqlNodeLimitEqual(t *testing.T) {
	cfg := CreateConfig()
	cfg.NodeLimit = 7

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
        posts {
          id
        }
      }
    }

    subscription PostFavSubscription($input: StoryLikeSubscribeInput) {
      postFavSubscribe(input: $input) {
        post {
          favers {
            count
          }
          favSentence {
            text
          }
        }
      }
    }

	  {
	    unnamed(truthyVal: true, falseyVal: false),
	    query
	  }
  `
	RunGraphqlLimitsTest(t, cfg, body, http.StatusOK)
}
