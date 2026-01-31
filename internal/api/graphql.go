// Package api provides the REST API server for Spartan Scraper.
//
// This file implements the GraphQL HTTP handler and playground endpoint.
// It wraps the graphql package to provide HTTP access to the GraphQL API.
//
// This file does NOT handle:
// - GraphQL schema definition (see internal/api/graphql/)
// - GraphQL resolvers (see internal/api/graphql/)
// - WebSocket subscriptions (handled separately)
//
// Invariants:
// - GraphQL endpoint is at /graphql
// - Playground is at /graphql/playground
// - Supports both GET and POST requests
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/api/graphql"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	graphqlgo "github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"
)

// GraphQLHandler wraps the GraphQL schema and provides HTTP handling.
type GraphQLHandler struct {
	schema  graphqlgo.Schema
	store   *store.Store
	manager *jobs.Manager
}

// NewGraphQLHandler creates a new GraphQL HTTP handler.
func NewGraphQLHandler(server *Server) (*GraphQLHandler, error) {
	resolver := graphql.NewResolver(server.store, server.manager)
	subscriptionManager := graphql.NewSubscriptionManager(server.manager)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Resolver:            resolver,
		SubscriptionManager: subscriptionManager,
	})
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create GraphQL schema", err)
	}

	return &GraphQLHandler{
		schema:  schema,
		store:   server.store,
		manager: server.manager,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (h *GraphQLHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Add resolver context for field resolution
	ctx := graphql.WithResolverContext(r.Context(), h.store, h.manager)
	r = r.WithContext(ctx)

	// Use the graphql-go handler
	hdl := handler.New(
		&handler.Config{
			Schema:   &h.schema,
			Pretty:   true,
			GraphiQL: false, // We provide our own playground
		},
	)

	hdl.ServeHTTP(w, r)
}

// graphQLRequest represents a GraphQL request body.
type graphQLRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// PlaygroundHandler serves the GraphQL Playground HTML page.
func PlaygroundHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(graphqlPlaygroundHTML))
}

// graphqlPlaygroundHTML is the HTML for the GraphQL Playground.
const graphqlPlaygroundHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>Spartan Scraper GraphQL Playground</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.26/build/static/css/index.css">
    <link rel="shortcut icon" href="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.26/build/favicon.png">
    <script src="https://cdn.jsdelivr.net/npm/graphql-playground-react@1.7.26/build/static/js/middleware.js"></script>
</head>
<body>
    <div id="root">
        <style>
            body {
                background-color: #172a3a;
                font-family: 'Open Sans', sans-serif;
                height: 100vh;
                margin: 0;
                overflow: hidden;
            }
            #root {
                height: 100%;
                width: 100%;
            }
            .loading {
                align-items: center;
                color: white;
                display: flex;
                flex-direction: column;
                height: 100%;
                justify-content: center;
                width: 100%;
            }
            .loading-title {
                font-size: 2rem;
                font-weight: 300;
                margin-bottom: 1rem;
            }
            .loading-subtitle {
                font-size: 1rem;
                opacity: 0.7;
            }
        </style>
        <div class="loading">
            <div class="loading-title">Spartan Scraper GraphQL Playground</div>
            <div class="loading-subtitle">Loading...</div>
        </div>
    </div>
    <script>
        window.addEventListener('load', function() {
            GraphQLPlayground.init(document.getElementById('root'), {
                endpoint: '/graphql',
                subscriptionEndpoint: '/v1/ws',
                settings: {
                    'request.credentials': 'same-origin'
                }
            });
        });
    </script>
</body>
</html>`

// handleGraphQL handles GraphQL queries and mutations.
func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if s.graphqlHandler == nil {
		writeError(w, r, apperrors.Internal("GraphQL handler not initialized"))
		return
	}

	s.graphqlHandler.ServeHTTP(w, r)
}

// handleGraphQLPlayground serves the GraphQL Playground.
func (s *Server) handleGraphQLPlayground(w http.ResponseWriter, r *http.Request) {
	PlaygroundHandler(w, r)
}
