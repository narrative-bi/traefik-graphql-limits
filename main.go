// Package traefikgraphqllimits provides a Traefik plugin to limit the depth of a GraphQL query
package traefikgraphqllimits

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

func buildGraphqlMaxDepthError(maxDepth, depthLimit int) string {
	errorBody := fmt.Sprintf(`{
    "errors": [
      {
        "code": 400,
        "message": "Query has depth of %d, which exceeds max depth of %d"
      }
    ] }`, maxDepth, depthLimit)

	return errorBody
}

func buildGraphqlBatchLimitError(batchCount, batchLimit int) string {
	errorBody := fmt.Sprintf(`{
    "errors": [
      {
        "code": 400,
        "message": "Query batch limit of %d, which exceeds limit of %d"
      }
    ] }`, batchCount, batchLimit)

	return errorBody
}

func buildGraphqlNodeLimitError(nodeCount, nodeLimit int) string {
	errorBody := fmt.Sprintf(`{
    "errors": [
      {
        "code": 400,
        "message": "Query node limit of %d, which exceeds limit of %d"
      }
    ] }`, nodeCount, nodeLimit)

	return errorBody
}

// QueryMetrics the query metrics for check.
type QueryMetrics struct {
	maxDepth   int
	batchCount int
	nodeCount  int
}

// CreateQueryMetrics creates the default query metrics.
func (queryMetrics QueryMetrics) CreateQueryMetrics() QueryMetrics {
	queryMetrics.maxDepth = 0
	queryMetrics.batchCount = 0
	queryMetrics.nodeCount = 0
	return queryMetrics
}

// Config the plugin configuration.
type Config struct {
	GraphQLPath string
	DepthLimit  int
	BatchLimit  int
	NodeLimit   int
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		GraphQLPath: "/graphql",
		DepthLimit:  0,
		BatchLimit:  0,
		NodeLimit:   0,
	}
}

// GraphqlLimit plugin configuration structure.
type GraphqlLimit struct {
	next        http.Handler
	name        string
	graphQLPath string
	depthLimit  int
	batchLimit  int
	nodeLimit   int
}

func calculateQueryMetrics(astDoc *ast.Document) QueryMetrics {
	queryMetrics := new(QueryMetrics).CreateQueryMetrics()

	v := &visitor.VisitorOptions{
		KindFuncMap: map[string]visitor.NamedVisitFuncs{
			kinds.SelectionSet: {
				Enter: func(p visitor.VisitFuncParams) (string, interface{}) {
					// NOTE: We do not calculate initial query depth here, so we start at -1
					depth := -1

					for _, element := range p.Path {
						if element == kinds.SelectionSet {
							depth++
						}
					}

					// NOTE: Top level query is start of new batch, otherwise it is a node
					if depth == 0 {
						queryMetrics.batchCount++
					} else {
						queryMetrics.nodeCount++
					}

					if depth > queryMetrics.maxDepth {
						queryMetrics.maxDepth = depth
					}

					return visitor.ActionNoChange, nil
				},
			},
		},
	}

	_ = visitor.Visit(astDoc, v, nil)

	return queryMetrics
}

// New created a new plugin.
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

func respondWithJSONError(rw http.ResponseWriter, json string) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusBadRequest)
	_, err := rw.Write([]byte(json))
	if err != nil {
		log.Printf("Error with response: %v", err)
	}
}

func isGraphqlRequest(req *http.Request, path string) bool {
	return req.Method == "POST" && req.URL.Path == path
}

func needToCheckLimits(depthLimit, batchLimit, nodeLimit int) bool {
	return depthLimit > 0 || batchLimit > 0 || nodeLimit > 0
}

func (d *GraphqlLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		respondWithJSONError(rw, errorBodyReadResponse)
		return
	}

	if isGraphqlRequest(req, d.graphQLPath) && needToCheckLimits(d.depthLimit, d.batchLimit, d.nodeLimit) {
		params := parser.ParseParams{
			Source: string(body),
			Options: parser.ParseOptions{
				NoLocation: true,
			},
		}

		parseResults, err := parser.Parse(params)
		if err != nil {
			respondWithJSONError(rw, errorGraphqlParsingResponse)
			return
		}

		queryMetrics := calculateQueryMetrics(parseResults)

		if d.depthLimit > 0 && queryMetrics.maxDepth > d.depthLimit {
			respondWithJSONError(rw, buildGraphqlMaxDepthError(queryMetrics.maxDepth, d.depthLimit))
			return
		}

		if d.batchLimit > 0 && queryMetrics.batchCount > d.batchLimit {
			respondWithJSONError(rw, buildGraphqlBatchLimitError(queryMetrics.batchCount, d.batchLimit))
		}

		if d.nodeLimit > 0 && queryMetrics.nodeCount > d.nodeLimit {
			respondWithJSONError(rw, buildGraphqlNodeLimitError(queryMetrics.nodeCount, d.nodeLimit))
		}
	}

	req.Body = io.NopCloser(bytes.NewBuffer(body))
	d.next.ServeHTTP(rw, req)
}
