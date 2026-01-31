// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file implements custom GraphQL scalars for JSON and Time types.
// It handles serialization and parsing of these types for GraphQL responses
// and inputs.
//
// This file does NOT handle:
// - Schema definition (see schema.go)
// - Resolver implementations (see resolvers.go)
// - Subscription handling (see subscriptions.go)
//
// Invariants:
// - JSON scalar accepts any valid JSON value
// - Time scalar uses RFC3339 format for serialization
package graphql

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// parseLiteral parses an AST value into a Go interface{}.
func parseLiteral(valueAST ast.Value) interface{} {
	switch v := valueAST.(type) {
	case *ast.StringValue:
		return v.Value
	case *ast.IntValue:
		return v.Value
	case *ast.FloatValue:
		return v.Value
	case *ast.BooleanValue:
		return v.Value
	case *ast.ObjectValue:
		result := make(map[string]interface{})
		for _, field := range v.Fields {
			result[field.Name.Value] = parseLiteral(field.Value)
		}
		return result
	case *ast.ListValue:
		result := make([]interface{}, len(v.Values))
		for i, val := range v.Values {
			result[i] = parseLiteral(val)
		}
		return result
	default:
		return nil
	}
}

// JSONScalar represents a JSON scalar type that can hold any valid JSON value.
var JSONScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "The JSON scalar type represents any valid JSON value.",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case map[string]interface{}:
			return v
		case []interface{}:
			return v
		case string:
			return v
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return v
		case bool:
			return v
		case nil:
			return nil
		case json.RawMessage:
			var result interface{}
			if err := json.Unmarshal(v, &result); err != nil {
				return nil
			}
			return result
		default:
			// Try to marshal/unmarshal to convert to map
			data, err := json.Marshal(v)
			if err != nil {
				return nil
			}
			var result interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				return nil
			}
			return result
		}
	},
	ParseValue: func(value interface{}) interface{} {
		switch v := value.(type) {
		case map[string]interface{}:
			return v
		case []interface{}:
			return v
		case string:
			return v
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return v
		case bool:
			return v
		case nil:
			return nil
		default:
			return nil
		}
	},
	ParseLiteral: parseLiteral,
})

// TimeScalar represents a time scalar type using RFC3339 format.
var TimeScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "Time",
	Description: "The Time scalar type represents a point in time using RFC3339 format.",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case time.Time:
			return v.Format(time.RFC3339)
		case *time.Time:
			if v == nil {
				return nil
			}
			return v.Format(time.RFC3339)
		case string:
			return v
		default:
			return nil
		}
	},
	ParseValue: func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil
			}
			return t
		case time.Time:
			return v
		default:
			return nil
		}
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch v := valueAST.(type) {
		case *ast.StringValue:
			t, err := time.Parse(time.RFC3339, v.Value)
			if err != nil {
				return nil
			}
			return t
		default:
			return nil
		}
	},
})

// parseTime parses a time value from various input formats.
func parseTime(value interface{}) (time.Time, error) {
	switch v := value.(type) {
	case string:
		return time.Parse(time.RFC3339, v)
	case time.Time:
		return v, nil
	default:
		return time.Time{}, fmt.Errorf("cannot parse time from %T", value)
	}
}
