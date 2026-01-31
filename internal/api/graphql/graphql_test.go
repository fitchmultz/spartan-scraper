// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file contains tests for the GraphQL schema, resolvers, and scalars.
package graphql

import (
	"os"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain initializes the GraphQL types before running tests.
func TestMain(m *testing.M) {
	// Initialize types once before all tests
	initTypes()
	os.Exit(m.Run())
}

// TestTypesInitialized verifies that all types are properly initialized.
func TestTypesInitialized(t *testing.T) {
	assert.NotNil(t, JobType, "JobType should not be nil")
	assert.NotNil(t, ChainType, "ChainType should not be nil")
	assert.NotNil(t, BatchType, "BatchType should not be nil")
	assert.NotNil(t, CrawlStateType, "CrawlStateType should not be nil")
	assert.NotNil(t, JobEdgeType, "JobEdgeType should not be nil")
	assert.NotNil(t, JobConnectionType, "JobConnectionType should not be nil")
	assert.NotNil(t, CrawlStateEdgeType, "CrawlStateEdgeType should not be nil")
	assert.NotNil(t, CrawlStateConnectionType, "CrawlStateConnectionType should not be nil")
}

// TestJSONScalar tests the JSON scalar type.
func TestJSONScalar(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "map",
			input:    map[string]interface{}{"key": "value", "num": 42},
			expected: map[string]interface{}{"key": "value", "num": 42},
		},
		{
			name:     "string",
			input:    "test string",
			expected: "test string",
		},
		{
			name:     "int",
			input:    42,
			expected: 42,
		},
		{
			name:     "float",
			input:    3.14,
			expected: 3.14,
		},
		{
			name:     "bool",
			input:    true,
			expected: true,
		},
		{
			name:     "nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONScalar.Serialize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTimeScalar tests the Time scalar type.
func TestTimeScalar(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "time",
			input:    now,
			expected: now.Format(time.RFC3339),
		},
		{
			name:     "string",
			input:    "2024-01-15T10:30:00Z",
			expected: "2024-01-15T10:30:00Z",
		},
		{
			name:     "nil pointer",
			input:    (*time.Time)(nil),
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeScalar.Serialize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTimeScalarParseValue tests parsing time values.
func TestTimeScalarParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected time.Time
		isNil    bool
	}{
		{
			name:     "valid RFC3339 string",
			input:    "2024-01-15T10:30:00Z",
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:     "time object",
			input:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			name:  "invalid string",
			input: "not a time",
			isNil: true,
		},
		{
			name:  "int",
			input: 42,
			isNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TimeScalar.ParseValue(tt.input)
			if tt.isNil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestJobKindEnum tests the JobKind enum.
func TestJobKindEnum(t *testing.T) {
	assert.NotNil(t, JobKindEnum)
	assert.Equal(t, "JobKind", JobKindEnum.Name())

	// Test enum values
	scrapeValue := JobKindEnum.Serialize(model.KindScrape)
	assert.Equal(t, "SCRAPE", scrapeValue)

	crawlValue := JobKindEnum.Serialize(model.KindCrawl)
	assert.Equal(t, "CRAWL", crawlValue)

	researchValue := JobKindEnum.Serialize(model.KindResearch)
	assert.Equal(t, "RESEARCH", researchValue)
}

// TestJobStatusEnum tests the JobStatus enum.
func TestJobStatusEnum(t *testing.T) {
	assert.NotNil(t, JobStatusEnum)
	assert.Equal(t, "JobStatus", JobStatusEnum.Name())

	// Test enum values
	queuedValue := JobStatusEnum.Serialize(model.StatusQueued)
	assert.Equal(t, "QUEUED", queuedValue)

	runningValue := JobStatusEnum.Serialize(model.StatusRunning)
	assert.Equal(t, "RUNNING", runningValue)

	succeededValue := JobStatusEnum.Serialize(model.StatusSucceeded)
	assert.Equal(t, "SUCCEEDED", succeededValue)

	failedValue := JobStatusEnum.Serialize(model.StatusFailed)
	assert.Equal(t, "FAILED", failedValue)

	canceledValue := JobStatusEnum.Serialize(model.StatusCanceled)
	assert.Equal(t, "CANCELED", canceledValue)
}

// TestSchemaCreation tests that the schema can be created.
func TestSchemaCreation(t *testing.T) {
	// Create a minimal resolver for testing
	resolver := &Resolver{}
	subscriptionManager := &SubscriptionManager{}

	schema, err := NewSchema(SchemaConfig{
		Resolver:            resolver,
		SubscriptionManager: subscriptionManager,
	})

	require.NoError(t, err)
	assert.NotNil(t, schema)

	// Verify query type exists
	queryType := schema.QueryType()
	assert.NotNil(t, queryType)
	assert.Equal(t, "Query", queryType.Name())

	// Verify mutation type exists
	mutationType := schema.MutationType()
	assert.NotNil(t, mutationType)
	assert.Equal(t, "Mutation", mutationType.Name())

	// Verify subscription type exists
	subscriptionType := schema.SubscriptionType()
	assert.NotNil(t, subscriptionType)
	assert.Equal(t, "Subscription", subscriptionType.Name())
}

// TestPageInfoType tests the PageInfo type.
func TestPageInfoType(t *testing.T) {
	assert.NotNil(t, PageInfoType)
	assert.Equal(t, "PageInfo", PageInfoType.Name())

	// Verify fields exist
	fields := PageInfoType.Fields()
	assert.NotNil(t, fields["hasNextPage"])
	assert.NotNil(t, fields["hasPreviousPage"])
	assert.NotNil(t, fields["startCursor"])
	assert.NotNil(t, fields["endCursor"])
	assert.NotNil(t, fields["totalCount"])
}

// TestJobTypeStructure tests the Job type structure.
func TestJobTypeStructure(t *testing.T) {
	assert.NotNil(t, JobType)
	assert.Equal(t, "Job", JobType.Name())

	// Verify fields exist
	fields := JobType.Fields()
	assert.NotNil(t, fields["id"])
	assert.NotNil(t, fields["kind"])
	assert.NotNil(t, fields["status"])
	assert.NotNil(t, fields["createdAt"])
	assert.NotNil(t, fields["updatedAt"])
	assert.NotNil(t, fields["params"])
	assert.NotNil(t, fields["resultPath"])
	assert.NotNil(t, fields["error"])
	assert.NotNil(t, fields["dependsOn"])
	assert.NotNil(t, fields["dependentJobs"])
	assert.NotNil(t, fields["dependencyStatus"])
	assert.NotNil(t, fields["chain"])
	assert.NotNil(t, fields["batch"])
}

// TestChainTypeStructure tests the Chain type structure.
func TestChainTypeStructure(t *testing.T) {
	assert.NotNil(t, ChainType)
	assert.Equal(t, "Chain", ChainType.Name())

	// Verify fields exist
	fields := ChainType.Fields()
	assert.NotNil(t, fields["id"])
	assert.NotNil(t, fields["name"])
	assert.NotNil(t, fields["description"])
	assert.NotNil(t, fields["definition"])
	assert.NotNil(t, fields["createdAt"])
	assert.NotNil(t, fields["updatedAt"])
	assert.NotNil(t, fields["jobs"])
}

// TestBatchTypeStructure tests the Batch type structure.
func TestBatchTypeStructure(t *testing.T) {
	assert.NotNil(t, BatchType)
	assert.Equal(t, "Batch", BatchType.Name())

	// Verify fields exist
	fields := BatchType.Fields()
	assert.NotNil(t, fields["id"])
	assert.NotNil(t, fields["kind"])
	assert.NotNil(t, fields["status"])
	assert.NotNil(t, fields["jobCount"])
	assert.NotNil(t, fields["createdAt"])
	assert.NotNil(t, fields["updatedAt"])
	assert.NotNil(t, fields["jobs"])
	assert.NotNil(t, fields["stats"])
}

// TestCrawlStateTypeStructure tests the CrawlState type structure.
func TestCrawlStateTypeStructure(t *testing.T) {
	assert.NotNil(t, CrawlStateType)
	assert.Equal(t, "CrawlState", CrawlStateType.Name())

	// Verify fields exist
	fields := CrawlStateType.Fields()
	assert.NotNil(t, fields["url"])
	assert.NotNil(t, fields["etag"])
	assert.NotNil(t, fields["lastModified"])
	assert.NotNil(t, fields["contentHash"])
	assert.NotNil(t, fields["lastScraped"])
	assert.NotNil(t, fields["depth"])
	assert.NotNil(t, fields["job"])
}

// TestMetricsSnapshotTypeStructure tests the MetricsSnapshot type structure.
func TestMetricsSnapshotTypeStructure(t *testing.T) {
	assert.NotNil(t, MetricsSnapshotType)
	assert.Equal(t, "MetricsSnapshot", MetricsSnapshotType.Name())

	// Verify fields exist
	fields := MetricsSnapshotType.Fields()
	assert.NotNil(t, fields["requestsPerSec"])
	assert.NotNil(t, fields["successRate"])
	assert.NotNil(t, fields["avgResponseTime"])
	assert.NotNil(t, fields["activeRequests"])
	assert.NotNil(t, fields["totalRequests"])
	assert.NotNil(t, fields["jobThroughput"])
	assert.NotNil(t, fields["avgJobDuration"])
	assert.NotNil(t, fields["timestamp"])
}

// TestJobConnectionTypeStructure tests the JobConnection type structure.
func TestJobConnectionTypeStructure(t *testing.T) {
	assert.NotNil(t, JobConnectionType)
	assert.Equal(t, "JobConnection", JobConnectionType.Name())

	// Verify fields exist
	fields := JobConnectionType.Fields()
	assert.NotNil(t, fields["edges"])
	assert.NotNil(t, fields["pageInfo"])
}

// TestBatchJobStatsTypeStructure tests the BatchJobStats type structure.
func TestBatchJobStatsTypeStructure(t *testing.T) {
	assert.NotNil(t, BatchJobStatsType)
	assert.Equal(t, "BatchJobStats", BatchJobStatsType.Name())

	// Verify fields exist
	fields := BatchJobStatsType.Fields()
	assert.NotNil(t, fields["queued"])
	assert.NotNil(t, fields["running"])
	assert.NotNil(t, fields["succeeded"])
	assert.NotNil(t, fields["failed"])
	assert.NotNil(t, fields["canceled"])
}

// TestChainDefinitionTypeStructure tests the ChainDefinition type structure.
func TestChainDefinitionTypeStructure(t *testing.T) {
	assert.NotNil(t, ChainDefinitionType)
	assert.Equal(t, "ChainDefinition", ChainDefinitionType.Name())

	// Verify fields exist
	fields := ChainDefinitionType.Fields()
	assert.NotNil(t, fields["nodes"])
	assert.NotNil(t, fields["edges"])
}

// TestChainNodeTypeStructure tests the ChainNode type structure.
func TestChainNodeTypeStructure(t *testing.T) {
	assert.NotNil(t, ChainNodeType)
	assert.Equal(t, "ChainNode", ChainNodeType.Name())

	// Verify fields exist
	fields := ChainNodeType.Fields()
	assert.NotNil(t, fields["id"])
	assert.NotNil(t, fields["kind"])
	assert.NotNil(t, fields["params"])
	assert.NotNil(t, fields["metadata"])
}

// TestChainEdgeTypeStructure tests the ChainEdge type structure.
func TestChainEdgeTypeStructure(t *testing.T) {
	assert.NotNil(t, ChainEdgeType)
	assert.Equal(t, "ChainEdge", ChainEdgeType.Name())

	// Verify fields exist
	fields := ChainEdgeType.Fields()
	assert.NotNil(t, fields["from"])
	assert.NotNil(t, fields["to"])
}

// TestChainMetadataTypeStructure tests the ChainMetadata type structure.
func TestChainMetadataTypeStructure(t *testing.T) {
	assert.NotNil(t, ChainMetadataType)
	assert.Equal(t, "ChainMetadata", ChainMetadataType.Name())

	// Verify fields exist
	fields := ChainMetadataType.Fields()
	assert.NotNil(t, fields["name"])
	assert.NotNil(t, fields["description"])
}

// TestCreateJobInputStructure tests the CreateJobInput type structure.
func TestCreateJobInputStructure(t *testing.T) {
	assert.NotNil(t, CreateJobInput)
	assert.Equal(t, "CreateJobInput", CreateJobInput.Name())

	// Verify fields exist
	fields := CreateJobInput.Fields()
	assert.NotNil(t, fields["kind"])
	assert.NotNil(t, fields["params"])
	assert.NotNil(t, fields["dependsOn"])
	assert.NotNil(t, fields["chainId"])
}

// TestCreateChainInputStructure tests the CreateChainInput type structure.
func TestCreateChainInputStructure(t *testing.T) {
	assert.NotNil(t, CreateChainInput)
	assert.Equal(t, "CreateChainInput", CreateChainInput.Name())

	// Verify fields exist
	fields := CreateChainInput.Fields()
	assert.NotNil(t, fields["name"])
	assert.NotNil(t, fields["description"])
	assert.NotNil(t, fields["definition"])
}

// TestDependencyStatusEnum tests the DependencyStatus enum.
func TestDependencyStatusEnum(t *testing.T) {
	assert.NotNil(t, DependencyStatusEnum)
	assert.Equal(t, "DependencyStatus", DependencyStatusEnum.Name())

	// Test enum values
	pendingValue := DependencyStatusEnum.Serialize(model.DependencyStatusPending)
	assert.Equal(t, "PENDING", pendingValue)

	readyValue := DependencyStatusEnum.Serialize(model.DependencyStatusReady)
	assert.Equal(t, "READY", readyValue)

	failedValue := DependencyStatusEnum.Serialize(model.DependencyStatusFailed)
	assert.Equal(t, "FAILED", failedValue)
}

// TestBatchStatusEnum tests the BatchStatus enum.
func TestBatchStatusEnum(t *testing.T) {
	assert.NotNil(t, BatchStatusEnum)
	assert.Equal(t, "BatchStatus", BatchStatusEnum.Name())

	// Test enum values
	pendingValue := BatchStatusEnum.Serialize(model.BatchStatusPending)
	assert.Equal(t, "PENDING", pendingValue)

	processingValue := BatchStatusEnum.Serialize(model.BatchStatusProcessing)
	assert.Equal(t, "PROCESSING", processingValue)

	completedValue := BatchStatusEnum.Serialize(model.BatchStatusCompleted)
	assert.Equal(t, "COMPLETED", completedValue)

	failedValue := BatchStatusEnum.Serialize(model.BatchStatusFailed)
	assert.Equal(t, "FAILED", failedValue)

	partialValue := BatchStatusEnum.Serialize(model.BatchStatusPartial)
	assert.Equal(t, "PARTIAL", partialValue)

	canceledValue := BatchStatusEnum.Serialize(model.BatchStatusCanceled)
	assert.Equal(t, "CANCELED", canceledValue)
}

// TestJobFilterInputStructure tests the JobFilterInput type structure.
func TestJobFilterInputStructure(t *testing.T) {
	assert.NotNil(t, JobFilterInput)
	assert.Equal(t, "JobFilter", JobFilterInput.Name())

	// Verify fields exist
	fields := JobFilterInput.Fields()
	assert.NotNil(t, fields["status"])
	assert.NotNil(t, fields["kind"])
	assert.NotNil(t, fields["chainId"])
	assert.NotNil(t, fields["batchId"])
}

// TestCrawlStateConnectionTypeStructure tests the CrawlStateConnection type structure.
func TestCrawlStateConnectionTypeStructure(t *testing.T) {
	assert.NotNil(t, CrawlStateConnectionType)
	assert.Equal(t, "CrawlStateConnection", CrawlStateConnectionType.Name())

	// Verify fields exist
	fields := CrawlStateConnectionType.Fields()
	assert.NotNil(t, fields["edges"])
	assert.NotNil(t, fields["pageInfo"])
}

// TestJobEdgeTypeStructure tests the JobEdge type structure.
func TestJobEdgeTypeStructure(t *testing.T) {
	assert.NotNil(t, JobEdgeType)
	assert.Equal(t, "JobEdge", JobEdgeType.Name())

	// Verify fields exist
	fields := JobEdgeType.Fields()
	assert.NotNil(t, fields["node"])
	assert.NotNil(t, fields["cursor"])
}

// TestCrawlStateEdgeTypeStructure tests the CrawlStateEdge type structure.
func TestCrawlStateEdgeTypeStructure(t *testing.T) {
	assert.NotNil(t, CrawlStateEdgeType)
	assert.Equal(t, "CrawlStateEdge", CrawlStateEdgeType.Name())

	// Verify fields exist
	fields := CrawlStateEdgeType.Fields()
	assert.NotNil(t, fields["node"])
	assert.NotNil(t, fields["cursor"])
}
