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

func buildGraphqlBatchLimitError(batchCount int, batchLimit int) string {
	errorBody := fmt.Sprintf(`{
    "errors": [
      {
        "code": 400,
        "message": "Query batch limit of %d, which exceeds limit of %d"
      }
    ] }`, batchCount, batchLimit)

	return errorBody
}

func buildGraphqlNodeLimitError(nodeCount int, nodeLimit int) string {
	errorBody := fmt.Sprintf(`{
    "errors": [
      {
        "code": 400,
        "message": "Query node limit of %d, which exceeds limit of %d"
      }
    ] }`, nodeCount, nodeLimit)

	return errorBody
}

type QueryMetrics struct {
	maxDepth   int
	batchCount int
	nodeCount  int
}

func (queryMetrics QueryMetrics) CreateQueryMetrics() QueryMetrics {
	queryMetrics.maxDepth = 0
	queryMetrics.batchCount = 0
	queryMetrics.nodeCount = 0
	return queryMetrics
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
							depth += 1
						}
					}

					// NOTE: Top level query is start of new batch, otherwise it is a node
					if depth == 0 {
						queryMetrics.batchCount += 1
					} else {
						queryMetrics.nodeCount += 1
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

			if err != nil {
				respondWithJson(rw, http.StatusBadRequest, errorGraphqlParsingResponse)
				return
			}

			queryMetrics := calculateQueryMetrics(parseResults)

			if d.depthLimit > 0 && queryMetrics.maxDepth > d.depthLimit {
				respondWithJson(rw, http.StatusBadRequest, buildGraphqlMaxDepthError(queryMetrics.maxDepth, d.depthLimit))
				return
			}

			if d.batchLimit > 0 && queryMetrics.batchCount > d.batchLimit {
				respondWithJson(rw, http.StatusBadRequest, buildGraphqlBatchLimitError(queryMetrics.batchCount, d.batchLimit))
			}

			if d.nodeLimit > 0 && queryMetrics.nodeCount > d.nodeLimit {
				respondWithJson(rw, http.StatusBadRequest, buildGraphqlNodeLimitError(queryMetrics.nodeCount, d.nodeLimit))
			}
		}
	}

	log.Printf("Pass through")

	req.Body = io.NopCloser(bytes.NewBuffer(body))
	d.next.ServeHTTP(rw, req)
}
