// Package paramdecode centralizes typed reads from persisted parameter maps.
//
// Purpose:
// - Provide one shared decoding path for job and scheduler parameter maps.
//
// Responsibilities:
// - Read scalar values with stable fallback behavior.
// - Decode JSON-shaped values into typed structs.
// - Recover persisted byte slices from JSON round trips.
//
// Scope:
// - Internal parameter coercion for persisted `map[string]any` payloads.
//
// Usage:
// - Used by jobs and scheduler when converting stored params into typed inputs.
//
// Invariants/Assumptions:
// - Missing or invalid values return explicit fallbacks instead of panicking.
// - Positive integer helpers treat zero and negative values as invalid.
// - Struct decoders accept either already-typed values or JSON-compatible maps.
package paramdecode

import (
	"encoding/base64"
	"encoding/json"
)

func String(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	value, _ := params[key].(string)
	return value
}

func Bool(params map[string]any, key string) bool {
	return BoolDefault(params, key, false)
}

func BoolDefault(params map[string]any, key string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}

func BoolValue(value any, fallback bool) bool {
	boolValue, ok := value.(bool)
	if !ok {
		return fallback
	}
	return boolValue
}

func PositiveInt(params map[string]any, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	return PositiveIntValue(params[key], fallback)
}

func PositiveIntValue(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		if typed <= 0 {
			return fallback
		}
		return typed
	case float64:
		if int(typed) <= 0 {
			return fallback
		}
		return int(typed)
	default:
		return fallback
	}
}

func StringSlice(params map[string]any, key string) []string {
	if params == nil {
		return nil
	}
	return StringSliceValue(params[key])
}

func StringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			stringValue, ok := item.(string)
			if ok {
				items = append(items, stringValue)
			}
		}
		return items
	default:
		return nil
	}
}

func Decode[T any](params map[string]any, key string) T {
	if params == nil {
		var zero T
		return zero
	}
	return DecodeValue[T](params[key])
}

func DecodeValue[T any](value any) T {
	var zero T
	if value == nil {
		return zero
	}
	if typed, ok := value.(T); ok {
		return typed
	}
	data, err := json.Marshal(value)
	if err != nil {
		return zero
	}
	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		return zero
	}
	return decoded
}

func DecodePtr[T any](params map[string]any, key string) *T {
	if params == nil {
		return nil
	}
	return DecodeValuePtr[T](params[key])
}

func DecodeValuePtr[T any](value any) *T {
	if value == nil {
		return nil
	}
	if typed, ok := value.(*T); ok {
		return typed
	}
	if typed, ok := value.(T); ok {
		return &typed
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil
	}
	return &decoded
}

func Bytes(params map[string]any, key string) []byte {
	if params == nil {
		return nil
	}
	return BytesValue(params[key])
}

func BytesValue(value any) []byte {
	if value == nil {
		return nil
	}
	if bytesValue, ok := value.([]byte); ok {
		return bytesValue
	}
	if stringValue, ok := value.(string); ok {
		if decoded, err := base64.StdEncoding.DecodeString(stringValue); err == nil {
			return decoded
		}
		return []byte(stringValue)
	}
	arrayValue, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]byte, 0, len(arrayValue))
	for _, item := range arrayValue {
		numberValue, ok := item.(float64)
		if ok {
			result = append(result, byte(numberValue))
		}
	}
	return result
}
