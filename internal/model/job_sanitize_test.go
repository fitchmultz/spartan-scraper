package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSanitizeJob_RemovesResultPath(t *testing.T) {
	job := Job{
		ID:         "test-123",
		Kind:       KindScrape,
		Status:     StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: "/Users/admin/.data/results/test-123.jsonl",
		Params:     map[string]interface{}{"url": "https://example.com"},
	}

	safe := SanitizeJob(job)

	if safe.ResultPath != "" {
		t.Errorf("ResultPath should be empty, got: %s", safe.ResultPath)
	}

	// Original should be unchanged
	if job.ResultPath == "" {
		t.Error("Original job ResultPath should not be modified")
	}
}

func TestSanitizeJob_RedactsSensitiveParams(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"url":      "https://example.com",
			"password": "secret123",
			"apiKey":   "abc-def-ghi",
			"token":    "bearer-token-xyz",
		},
	}

	safe := SanitizeJob(job)

	if safe.Params["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted, got: %v", safe.Params["password"])
	}
	if safe.Params["apiKey"] != "[REDACTED]" {
		t.Errorf("apiKey should be redacted, got: %v", safe.Params["apiKey"])
	}
	if safe.Params["token"] != "[REDACTED]" {
		t.Errorf("token should be redacted, got: %v", safe.Params["token"])
	}
	if safe.Params["url"] != "https://example.com" {
		t.Errorf("url should not be redacted, got: %v", safe.Params["url"])
	}
}

func TestSanitizeJob_PreservesNonSensitiveKeys(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"authProfile":    "should-be-preserved",
			"keyboardLayout": "should-be-preserved",
			"monkeyId":       "should-be-preserved",
			"keyframeRate":   "should-be-preserved",
			"authorization":  "should-be-redacted",
			"api_key":        "should-be-redacted",
			"user_password":  "should-be-redacted",
		},
	}

	safe := SanitizeJob(job)

	// These should NOT be redacted (contain but don't match sensitive patterns)
	if safe.Params["authProfile"] != "should-be-preserved" {
		t.Errorf("authProfile should be preserved, got: %v", safe.Params["authProfile"])
	}
	if safe.Params["keyboardLayout"] != "should-be-preserved" {
		t.Errorf("keyboardLayout should be preserved, got: %v", safe.Params["keyboardLayout"])
	}
	if safe.Params["monkeyId"] != "should-be-preserved" {
		t.Errorf("monkeyId should be preserved, got: %v", safe.Params["monkeyId"])
	}
	if safe.Params["keyframeRate"] != "should-be-preserved" {
		t.Errorf("keyframeRate should be preserved, got: %v", safe.Params["keyframeRate"])
	}

	// These SHOULD be redacted (exact match or proper pattern)
	if safe.Params["authorization"] != "[REDACTED]" {
		t.Errorf("authorization should be redacted, got: %v", safe.Params["authorization"])
	}
	if safe.Params["api_key"] != "[REDACTED]" {
		t.Errorf("api_key should be redacted, got: %v", safe.Params["api_key"])
	}
	if safe.Params["user_password"] != "[REDACTED]" {
		t.Errorf("user_password should be redacted, got: %v", safe.Params["user_password"])
	}
}

func TestSanitizeJob_RedactsNestedParams(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindCrawl,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"url": "https://example.com",
			"auth": map[string]interface{}{
				"username": "admin",
				"password": "secret123",
			},
			"config": map[string]interface{}{
				"apiKey": "nested-key",
				"nested": map[string]interface{}{
					"secret": "deep-secret",
				},
			},
		},
	}

	safe := SanitizeJob(job)

	// Top-level auth should be redacted
	if safe.Params["auth"] != "[REDACTED]" {
		t.Errorf("auth should be redacted at top level, got: %v", safe.Params["auth"])
	}

	// Nested apiKey should be redacted
	config := safe.Params["config"].(map[string]interface{})
	if config["apiKey"] != "[REDACTED]" {
		t.Errorf("nested apiKey should be redacted, got: %v", config["apiKey"])
	}

	// Deeply nested secret should be redacted
	nested := config["nested"].(map[string]interface{})
	if nested["secret"] != "[REDACTED]" {
		t.Errorf("deeply nested secret should be redacted, got: %v", nested["secret"])
	}
}

func TestSanitizeJob_RedactsHeaders(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"headers": map[string]interface{}{
				"Authorization":       "Bearer secret-token",
				"Cookie":              "session=abc123",
				"Proxy-Authorization": "Basic encoded",
				"X-Access-Token":      "access-token-xyz",
				"X-Token":             "token-abc",
				"Content-Type":        "application/json",
				"X-Custom":            "custom-value",
			},
		},
	}

	safe := SanitizeJob(job)

	headers := safe.Params["headers"].(map[string]interface{})

	if headers["Authorization"] != "[REDACTED]" {
		t.Errorf("Authorization header should be redacted, got: %v", headers["Authorization"])
	}
	if headers["Cookie"] != "[REDACTED]" {
		t.Errorf("Cookie header should be redacted, got: %v", headers["Cookie"])
	}
	if headers["Proxy-Authorization"] != "[REDACTED]" {
		t.Errorf("Proxy-Authorization header should be redacted, got: %v", headers["Proxy-Authorization"])
	}
	if headers["X-Access-Token"] != "[REDACTED]" {
		t.Errorf("X-Access-Token header should be redacted, got: %v", headers["X-Access-Token"])
	}
	if headers["X-Token"] != "[REDACTED]" {
		t.Errorf("X-Token header should be redacted, got: %v", headers["X-Token"])
	}
	if headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header should not be redacted, got: %v", headers["Content-Type"])
	}
	if headers["X-Custom"] != "custom-value" {
		t.Errorf("X-Custom header should not be redacted, got: %v", headers["X-Custom"])
	}
}

func TestSanitizeJob_RedactsHeaderArray(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"headers": []interface{}{
				"Authorization: Bearer secret-token",
				"Cookie: session=abc123",
				"Content-Type: application/json",
			},
		},
	}

	safe := SanitizeJob(job)

	headers := safe.Params["headers"].([]interface{})

	if headers[0] != "Authorization: [REDACTED]" {
		t.Errorf("Authorization header should be redacted, got: %v", headers[0])
	}
	if headers[1] != "Cookie: [REDACTED]" {
		t.Errorf("Cookie header should be redacted, got: %v", headers[1])
	}
	if headers[2] != "Content-Type: application/json" {
		t.Errorf("Content-Type header should not be redacted, got: %v", headers[2])
	}
}

func TestSanitizeJob_RedactsPathsInError(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusFailed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Error:     "Failed to write to /Users/admin/.data/results/test.jsonl: permission denied",
	}

	safe := SanitizeJob(job)

	if strings.Contains(safe.Error, "/Users/admin/.data/results") {
		t.Errorf("Error should not contain filesystem path, got: %s", safe.Error)
	}
	if !strings.Contains(safe.Error, "[REDACTED]") {
		t.Errorf("Error should contain [REDACTED] placeholder, got: %s", safe.Error)
	}
}

func TestSanitizeJob_RedactsPathsInParams(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"outputPath":       "/home/user/output.json",
			"fileUrl":          "file:///Users/admin/secret.txt",
			"windowsBackslash": "C:\\Users\\Alice\\Documents\\secret.txt",
			"windowsForward":   "D:/Data/Results/output.json",
		},
	}

	safe := SanitizeJob(job)

	if !strings.Contains(safe.Params["outputPath"].(string), "[REDACTED]") {
		t.Errorf("outputPath should be redacted, got: %v", safe.Params["outputPath"])
	}
	if !strings.Contains(safe.Params["fileUrl"].(string), "[REDACTED]") {
		t.Errorf("fileUrl should be redacted, got: %v", safe.Params["fileUrl"])
	}
	if !strings.Contains(safe.Params["windowsBackslash"].(string), "[REDACTED]") {
		t.Errorf("windowsBackslash should be redacted, got: %v", safe.Params["windowsBackslash"])
	}
	if !strings.Contains(safe.Params["windowsForward"].(string), "[REDACTED]") {
		t.Errorf("windowsForward should be redacted, got: %v", safe.Params["windowsForward"])
	}
}

func TestSanitizeJobs(t *testing.T) {
	jobs := []Job{
		{
			ID:         "job-1",
			Kind:       KindScrape,
			Status:     StatusSucceeded,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			ResultPath: "/path/to/result1.jsonl",
			Params:     map[string]interface{}{"password": "secret1"},
		},
		{
			ID:         "job-2",
			Kind:       KindCrawl,
			Status:     StatusFailed,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			ResultPath: "/path/to/result2.jsonl",
			Params:     map[string]interface{}{"apiKey": "secret2"},
		},
	}

	safe := SanitizeJobs(jobs)

	if len(safe) != 2 {
		t.Fatalf("Expected 2 jobs, got %d", len(safe))
	}

	for i, job := range safe {
		if job.ResultPath != "" {
			t.Errorf("Job %d: ResultPath should be empty", i)
		}
	}

	if safe[0].Params["password"] != "[REDACTED]" {
		t.Errorf("Job 0: password should be redacted")
	}
	if safe[1].Params["apiKey"] != "[REDACTED]" {
		t.Errorf("Job 1: apiKey should be redacted")
	}

	// Originals should be unchanged
	if jobs[0].ResultPath == "" || jobs[1].ResultPath == "" {
		t.Error("Original jobs should not be modified")
	}
}

func TestSanitizeJob_NilParams(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params:    nil,
	}

	safe := SanitizeJob(job)

	if safe.Params != nil {
		t.Errorf("Params should be nil, got: %v", safe.Params)
	}
}

func TestSanitizeJob_JSONSerialization(t *testing.T) {
	job := Job{
		ID:         "test-123",
		Kind:       KindScrape,
		Status:     StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: "/secret/path/result.jsonl",
		Params: map[string]interface{}{
			"url":      "https://example.com",
			"password": "secret123",
		},
		Error: "Failed at /Users/admin/secret/file.txt",
	}

	safe := SanitizeJob(job)

	// Serialize to JSON
	data, err := json.Marshal(safe)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Verify ResultPath is not in JSON (due to omitempty)
	if strings.Contains(jsonStr, "resultPath") {
		t.Errorf("JSON should not contain resultPath field, got: %s", jsonStr)
	}

	// Verify sensitive data is redacted
	if strings.Contains(jsonStr, "secret123") {
		t.Errorf("JSON should not contain password, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "/secret/path") {
		t.Errorf("JSON should not contain filesystem path, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "/Users/admin") {
		t.Errorf("JSON should not contain user path in error, got: %s", jsonStr)
	}
}

func TestSanitizeJob_PreservesNonSensitiveData(t *testing.T) {
	created := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	updated := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC)

	job := Job{
		ID:         "test-123",
		Kind:       KindResearch,
		Status:     StatusRunning,
		CreatedAt:  created,
		UpdatedAt:  updated,
		ResultPath: "/secret/path",
		Params: map[string]interface{}{
			"url":      "https://example.com",
			"maxDepth": 3,
			"maxPages": 100,
			"headless": true,
			"query":    "test query",
			"password": "should-be-redacted",
		},
	}

	safe := SanitizeJob(job)

	if safe.ID != "test-123" {
		t.Errorf("ID should be preserved, got: %s", safe.ID)
	}
	if safe.Kind != KindResearch {
		t.Errorf("Kind should be preserved, got: %s", safe.Kind)
	}
	if safe.Status != StatusRunning {
		t.Errorf("Status should be preserved, got: %s", safe.Status)
	}
	if !safe.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt should be preserved, got: %v", safe.CreatedAt)
	}
	if !safe.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt should be preserved, got: %v", safe.UpdatedAt)
	}

	// Non-sensitive params should be preserved
	if safe.Params["url"] != "https://example.com" {
		t.Errorf("url should be preserved, got: %v", safe.Params["url"])
	}
	if safe.Params["maxDepth"] != 3 {
		t.Errorf("maxDepth should be preserved, got: %v", safe.Params["maxDepth"])
	}
	if safe.Params["maxPages"] != 100 {
		t.Errorf("maxPages should be preserved, got: %v", safe.Params["maxPages"])
	}
	if safe.Params["headless"] != true {
		t.Errorf("headless should be preserved, got: %v", safe.Params["headless"])
	}
	if safe.Params["query"] != "test query" {
		t.Errorf("query should be preserved, got: %v", safe.Params["query"])
	}

	// Sensitive params should be redacted
	if safe.Params["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted, got: %v", safe.Params["password"])
	}
}

func TestSanitizeJob_CaseInsensitiveKeyMatching(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"PASSWORD":     "uppercase",
			"ApiKey":       "mixed-case",
			"AUTH_TOKEN":   "underscore",
			"userPassword": "contained",
			"myApiKey":     "prefixed",
		},
	}

	safe := SanitizeJob(job)

	if safe.Params["PASSWORD"] != "[REDACTED]" {
		t.Errorf("PASSWORD (uppercase) should be redacted, got: %v", safe.Params["PASSWORD"])
	}
	if safe.Params["ApiKey"] != "[REDACTED]" {
		t.Errorf("ApiKey (mixed-case) should be redacted, got: %v", safe.Params["ApiKey"])
	}
	if safe.Params["AUTH_TOKEN"] != "[REDACTED]" {
		t.Errorf("AUTH_TOKEN should be redacted, got: %v", safe.Params["AUTH_TOKEN"])
	}
	if safe.Params["userPassword"] != "[REDACTED]" {
		t.Errorf("userPassword (containing 'password') should be redacted, got: %v", safe.Params["userPassword"])
	}
	if safe.Params["myApiKey"] != "[REDACTED]" {
		t.Errorf("myApiKey (containing 'apikey') should be redacted, got: %v", safe.Params["myApiKey"])
	}
}

func TestSanitizeJob_SliceInParams(t *testing.T) {
	job := Job{
		ID:        "test-123",
		Kind:      KindScrape,
		Status:    StatusRunning,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Params: map[string]interface{}{
			"urls": []interface{}{
				"https://example.com",
				"https://test.com",
			},
			"nested": []interface{}{
				map[string]interface{}{
					"secret": "should-be-redacted",
				},
				"plain-string",
			},
		},
	}

	safe := SanitizeJob(job)

	urls := safe.Params["urls"].([]interface{})
	if len(urls) != 2 {
		t.Fatalf("Expected 2 URLs, got %d", len(urls))
	}
	if urls[0] != "https://example.com" {
		t.Errorf("First URL should be preserved, got: %v", urls[0])
	}

	nested := safe.Params["nested"].([]interface{})
	if len(nested) != 2 {
		t.Fatalf("Expected 2 nested items, got %d", len(nested))
	}

	nestedMap := nested[0].(map[string]interface{})
	if nestedMap["secret"] != "[REDACTED]" {
		t.Errorf("Nested secret should be redacted, got: %v", nestedMap["secret"])
	}
}
