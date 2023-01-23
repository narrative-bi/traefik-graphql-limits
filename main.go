package traefik_graphql_limits

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/visitor"
)

const errorBodyReadResponse = `{
  "errors": [
    {
      "code": 400,
      "message": "Failed to read request body"
    }
  ]
}`

const errorGraphqlParsingResponse = `{
  "errors": [
    {
      "code": 400,
      "message": "Not a valid graphql query"
    }
  ]
}`

func buildGraphqlMaxDepthError(maxDepth int, depthLimit int) string {
  errorBody := fmt.Sprintf(`{
    "errors": [
      {
        "code": 400,
        "message": "Query has depth of %d, which exceeds max depth of %d"
      }
    ] }`, maxDepth, depthLimit)

  return errorBody
}

type Config struct {
	GraphQLPath string
	DepthLimit  int
	BatchLimit  int
	NodeLimit   int
}

func CreateConfig() *Config {
	return &Config{
		GraphQLPath: "/graphql",
		DepthLimit:  0,
		BatchLimit:  0,
		NodeLimit:   0,
	}
}

type GraphqlLimit struct {
	next        http.Handler
	name        string
	graphQLPath string
	depthLimit  int
	batchLimit  int
	nodeLimit   int
}

func calculateMaxDepth(astDoc *ast.Document) int {
	maxDepth := 0

	v := &visitor.VisitorOptions{
		KindFuncMap: map[string]visitor.NamedVisitFuncs{
			kinds.SelectionSet: {
				Enter: func(p visitor.VisitFuncParams) (string, interface{}) {
					depth := 0

					for _, element := range p.Path {
						if element == kinds.SelectionSet {
							depth += 1
						}
					}

					if depth > maxDepth {
						maxDepth = depth
					}

					return visitor.ActionNoChange, nil
				},
			},
		},
	}

	_ = visitor.Visit(astDoc, v, nil)

	return maxDepth
}

func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &GraphqlLimit{
		next:        next,
		name:        name,
		graphQLPath: config.GraphQLPath,
		depthLimit:  config.DepthLimit,
		batchLimit:  config.BatchLimit,
		nodeLimit:   config.NodeLimit,
	}, nil
}

func respondWithJson(rw http.ResponseWriter, statusCode int, json string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(statusCode)
	_, err := rw.Write([]byte(json))
	if err != nil {
		log.Printf("Error with response: %v", err)
	}
}

func (d *GraphqlLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)

	if err != nil {
		log.Printf("Error reading body: %v", err)
		respondWithJson(rw, http.StatusBadRequest, errorBodyReadResponse)
		return
	}

	if req.Method == "POST" && req.URL.Path == d.graphQLPath {
		log.Printf("Checking graphql query")

		if d.depthLimit > 0 || d.batchLimit > 0 || d.nodeLimit > 0 {
			params := parser.ParseParams{
				Source: string(body),
			}

			parseResults, err := parser.Parse(params)

			// log.Printf("Parse results: %v", parseResults)
			// log.Printf("Error parsing query: %v", err)

			if err != nil {
				// log.Printf("Error parsing query: %v", err)
				respondWithJson(rw, http.StatusBadRequest, errorGraphqlParsingResponse)
				return
			}

			if d.depthLimit > 0 {
				maxDepth := calculateMaxDepth(parseResults)

				if maxDepth > d.depthLimit {
					respondWithJson(rw, http.StatusBadRequest, buildGraphqlMaxDepthError(maxDepth, d.depthLimit))
					return
				}
			}

			if d.batchLimit > 0 {
				// log.Printf("Batch limit is set to %d", d.depthLimit)
				respondWithJson(rw, http.StatusBadRequest, errorBodyReadResponse)
			}

			if d.nodeLimit > 0 {
				// log.Printf("Node limit is set to %d", d.depthLimit)
				respondWithJson(rw, http.StatusBadRequest, errorBodyReadResponse)
			}
		}
	}

	log.Printf("Pass through")

	req.Body = io.NopCloser(bytes.NewBuffer(body))
	d.next.ServeHTTP(rw, req)
}
