// Package graphql provides GraphQL API support for Spartan Scraper.
//
// This file defines the GraphQL schema including types, enums, inputs,
// and the root query, mutation, and subscription types.
//
// This file does NOT handle:
// - Resolver implementations (see resolvers.go)
// - Custom scalar serialization (see scalars.go)
// - Subscription handling (see subscriptions.go)
//
// Invariants:
// - All types map to domain models in internal/model
// - Field resolvers handle relationship resolution
// - Pagination follows Relay Connection spec pattern
package graphql

import (
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/graphql-go/graphql"
)

// JobKindEnum represents the job kind enumeration.
var JobKindEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "JobKind",
	Description: "The kind of job (scrape, crawl, research)",
	Values: graphql.EnumValueConfigMap{
		"SCRAPE":   {Value: model.KindScrape},
		"CRAWL":    {Value: model.KindCrawl},
		"RESEARCH": {Value: model.KindResearch},
	},
})

// JobStatusEnum represents the job status enumeration.
var JobStatusEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "JobStatus",
	Description: "The status of a job",
	Values: graphql.EnumValueConfigMap{
		"QUEUED":    {Value: model.StatusQueued},
		"RUNNING":   {Value: model.StatusRunning},
		"SUCCEEDED": {Value: model.StatusSucceeded},
		"FAILED":    {Value: model.StatusFailed},
		"CANCELED":  {Value: model.StatusCanceled},
	},
})

// DependencyStatusEnum represents the dependency status enumeration.
var DependencyStatusEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "DependencyStatus",
	Description: "The status of a job's dependencies",
	Values: graphql.EnumValueConfigMap{
		"PENDING": {Value: model.DependencyStatusPending},
		"READY":   {Value: model.DependencyStatusReady},
		"FAILED":  {Value: model.DependencyStatusFailed},
	},
})

// BatchStatusEnum represents the batch status enumeration.
var BatchStatusEnum = graphql.NewEnum(graphql.EnumConfig{
	Name:        "BatchStatus",
	Description: "The aggregate status of a batch",
	Values: graphql.EnumValueConfigMap{
		"PENDING":    {Value: model.BatchStatusPending},
		"PROCESSING": {Value: model.BatchStatusProcessing},
		"COMPLETED":  {Value: model.BatchStatusCompleted},
		"FAILED":     {Value: model.BatchStatusFailed},
		"PARTIAL":    {Value: model.BatchStatusPartial},
		"CANCELED":   {Value: model.BatchStatusCanceled},
	},
})

// PageInfoType represents pagination information.
var PageInfoType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "PageInfo",
	Description: "Pagination information for connections",
	Fields: graphql.Fields{
		"hasNextPage":     {Type: graphql.NewNonNull(graphql.Boolean)},
		"hasPreviousPage": {Type: graphql.NewNonNull(graphql.Boolean)},
		"startCursor":     {Type: graphql.String},
		"endCursor":       {Type: graphql.String},
		"totalCount":      {Type: graphql.NewNonNull(graphql.Int)},
	},
})

// ChainMetadataType represents metadata for a chain node.
var ChainMetadataType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "ChainMetadata",
	Description: "Human-readable information about a chain node",
	Fields: graphql.Fields{
		"name":        {Type: graphql.String},
		"description": {Type: graphql.String},
	},
})

// ChainNodeType represents a node in a chain definition.
var ChainNodeType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "ChainNode",
	Description: "A job template within a chain",
	Fields: graphql.Fields{
		"id":       {Type: graphql.NewNonNull(graphql.ID)},
		"kind":     {Type: graphql.NewNonNull(JobKindEnum)},
		"params":   {Type: JSONScalar},
		"metadata": {Type: ChainMetadataType},
	},
})

// ChainEdgeType represents an edge in a chain definition.
var ChainEdgeType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "ChainEdge",
	Description: "A dependency relationship between chain nodes",
	Fields: graphql.Fields{
		"from": {Type: graphql.NewNonNull(graphql.ID)},
		"to":   {Type: graphql.NewNonNull(graphql.ID)},
	},
})

// ChainDefinitionType represents the definition of a chain.
var ChainDefinitionType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "ChainDefinition",
	Description: "The workflow structure of a chain as a DAG",
	Fields: graphql.Fields{
		"nodes": {Type: graphql.NewList(graphql.NewNonNull(ChainNodeType))},
		"edges": {Type: graphql.NewList(graphql.NewNonNull(ChainEdgeType))},
	},
})

// BatchJobStatsType represents statistics for a batch.
var BatchJobStatsType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "BatchJobStats",
	Description: "Aggregated job status counts for a batch",
	Fields: graphql.Fields{
		"queued":    {Type: graphql.NewNonNull(graphql.Int)},
		"running":   {Type: graphql.NewNonNull(graphql.Int)},
		"succeeded": {Type: graphql.NewNonNull(graphql.Int)},
		"failed":    {Type: graphql.NewNonNull(graphql.Int)},
		"canceled":  {Type: graphql.NewNonNull(graphql.Int)},
	},
})

// MetricsSnapshotType represents a point-in-time metrics snapshot.
var MetricsSnapshotType = graphql.NewObject(graphql.ObjectConfig{
	Name:        "MetricsSnapshot",
	Description: "Current system metrics",
	Fields: graphql.Fields{
		"requestsPerSec":  {Type: graphql.NewNonNull(graphql.Float)},
		"successRate":     {Type: graphql.NewNonNull(graphql.Float)},
		"avgResponseTime": {Type: graphql.NewNonNull(graphql.Float)},
		"activeRequests":  {Type: graphql.NewNonNull(graphql.Int)},
		"totalRequests":   {Type: graphql.NewNonNull(graphql.Int)},
		"jobThroughput":   {Type: graphql.NewNonNull(graphql.Float)},
		"avgJobDuration":  {Type: graphql.NewNonNull(graphql.Float)},
		"timestamp":       {Type: graphql.NewNonNull(TimeScalar)},
	},
})

// JobFilterInput represents filter options for jobs.
var JobFilterInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name:        "JobFilter",
	Description: "Filter options for jobs",
	Fields: graphql.InputObjectConfigFieldMap{
		"status":  {Type: JobStatusEnum},
		"kind":    {Type: JobKindEnum},
		"chainId": {Type: graphql.ID},
		"batchId": {Type: graphql.ID},
	},
})

// CreateJobInput represents input for creating a job.
var CreateJobInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name:        "CreateJobInput",
	Description: "Input for creating a new job",
	Fields: graphql.InputObjectConfigFieldMap{
		"kind":      {Type: graphql.NewNonNull(JobKindEnum)},
		"params":    {Type: JSONScalar},
		"dependsOn": {Type: graphql.NewList(graphql.NewNonNull(graphql.ID))},
		"chainId":   {Type: graphql.ID},
	},
})

// CreateChainInput represents input for creating a chain.
var CreateChainInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name:        "CreateChainInput",
	Description: "Input for creating a new chain",
	Fields: graphql.InputObjectConfigFieldMap{
		"name":        {Type: graphql.NewNonNull(graphql.String)},
		"description": {Type: graphql.String},
		"definition":  {Type: graphql.NewNonNull(JSONScalar)},
	},
})

// JobType represents a job in the system.
var JobType *graphql.Object

// ChainType represents a job chain.
var ChainType *graphql.Object

// BatchType represents a batch of jobs.
var BatchType *graphql.Object

// CrawlStateType represents the state of a crawled URL.
var CrawlStateType *graphql.Object

// JobEdgeType represents an edge in a job connection.
var JobEdgeType *graphql.Object

// JobConnectionType represents a paginated list of jobs.
var JobConnectionType *graphql.Object

// CrawlStateEdgeType represents an edge in a crawl state connection.
var CrawlStateEdgeType *graphql.Object

// CrawlStateConnectionType represents a paginated list of crawl states.
var CrawlStateConnectionType *graphql.Object

// initTypes initializes all GraphQL types to avoid initialization cycles.
// This function is idempotent and safe to call multiple times.
func initTypes() {
	// JobEdgeType - only create if not already initialized
	if JobEdgeType == nil {
		JobEdgeType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "JobEdge",
			Description: "An edge in a job connection",
			Fields: graphql.FieldsThunk(func() graphql.Fields {
				return graphql.Fields{
					"node":   {Type: graphql.NewNonNull(JobType)},
					"cursor": {Type: graphql.NewNonNull(graphql.String)},
				}
			}),
		})
	}

	// JobConnectionType
	if JobConnectionType == nil {
		JobConnectionType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "JobConnection",
			Description: "A paginated list of jobs",
			Fields: graphql.Fields{
				"edges":    {Type: graphql.NewList(graphql.NewNonNull(JobEdgeType))},
				"pageInfo": {Type: graphql.NewNonNull(PageInfoType)},
			},
		})
	}

	// CrawlStateEdgeType
	if CrawlStateEdgeType == nil {
		CrawlStateEdgeType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "CrawlStateEdge",
			Description: "An edge in a crawl state connection",
			Fields: graphql.FieldsThunk(func() graphql.Fields {
				return graphql.Fields{
					"node":   {Type: graphql.NewNonNull(CrawlStateType)},
					"cursor": {Type: graphql.NewNonNull(graphql.String)},
				}
			}),
		})
	}

	// CrawlStateConnectionType
	if CrawlStateConnectionType == nil {
		CrawlStateConnectionType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "CrawlStateConnection",
			Description: "A paginated list of crawl states",
			Fields: graphql.Fields{
				"edges":    {Type: graphql.NewList(graphql.NewNonNull(CrawlStateEdgeType))},
				"pageInfo": {Type: graphql.NewNonNull(PageInfoType)},
			},
		})
	}

	// CrawlStateType
	if CrawlStateType == nil {
		CrawlStateType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "CrawlState",
			Description: "The state of a URL for incremental crawling",
			Fields: graphql.FieldsThunk(func() graphql.Fields {
				return graphql.Fields{
					"url":          {Type: graphql.NewNonNull(graphql.String)},
					"etag":         {Type: graphql.String},
					"lastModified": {Type: graphql.String},
					"contentHash":  {Type: graphql.String},
					"lastScraped":  {Type: graphql.NewNonNull(TimeScalar)},
					"depth":        {Type: graphql.NewNonNull(graphql.Int)},
					"job": {
						Type:    JobType,
						Resolve: resolveCrawlStateJob,
					},
				}
			}),
		})
	}

	// BatchType
	if BatchType == nil {
		BatchType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "Batch",
			Description: "A collection of related jobs submitted together",
			Fields: graphql.FieldsThunk(func() graphql.Fields {
				return graphql.Fields{
					"id":        {Type: graphql.NewNonNull(graphql.ID)},
					"kind":      {Type: graphql.NewNonNull(JobKindEnum)},
					"status":    {Type: graphql.NewNonNull(BatchStatusEnum)},
					"jobCount":  {Type: graphql.NewNonNull(graphql.Int)},
					"createdAt": {Type: graphql.NewNonNull(TimeScalar)},
					"updatedAt": {Type: graphql.NewNonNull(TimeScalar)},
					"jobs": {
						Type:    graphql.NewList(graphql.NewNonNull(JobType)),
						Resolve: resolveBatchJobs,
					},
					"stats": {
						Type:    graphql.NewNonNull(BatchJobStatsType),
						Resolve: resolveBatchStats,
					},
				}
			}),
		})
	}

	// ChainType
	if ChainType == nil {
		ChainType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "Chain",
			Description: "A named, reusable workflow definition",
			Fields: graphql.FieldsThunk(func() graphql.Fields {
				return graphql.Fields{
					"id":          {Type: graphql.NewNonNull(graphql.ID)},
					"name":        {Type: graphql.NewNonNull(graphql.String)},
					"description": {Type: graphql.String},
					"definition":  {Type: graphql.NewNonNull(ChainDefinitionType)},
					"createdAt":   {Type: graphql.NewNonNull(TimeScalar)},
					"updatedAt":   {Type: graphql.NewNonNull(TimeScalar)},
					"jobs": {
						Type:    graphql.NewList(graphql.NewNonNull(JobType)),
						Resolve: resolveChainJobs,
					},
				}
			}),
		})
	}

	// JobType
	if JobType == nil {
		JobType = graphql.NewObject(graphql.ObjectConfig{
			Name:        "Job",
			Description: "A scraping, crawling, or research job",
			Fields: graphql.FieldsThunk(func() graphql.Fields {
				return graphql.Fields{
					"id":         {Type: graphql.NewNonNull(graphql.ID)},
					"kind":       {Type: graphql.NewNonNull(JobKindEnum)},
					"status":     {Type: graphql.NewNonNull(JobStatusEnum)},
					"createdAt":  {Type: graphql.NewNonNull(TimeScalar)},
					"updatedAt":  {Type: graphql.NewNonNull(TimeScalar)},
					"params":     {Type: JSONScalar},
					"resultPath": {Type: graphql.String},
					"error":      {Type: graphql.String},
					"dependsOn": {
						Type:    graphql.NewList(graphql.NewNonNull(JobType)),
						Resolve: resolveJobDependsOn,
					},
					"dependentJobs": {
						Type:    graphql.NewList(graphql.NewNonNull(JobType)),
						Resolve: resolveJobDependentJobs,
					},
					"dependencyStatus": {Type: DependencyStatusEnum},
					"chain": {
						Type:    ChainType,
						Resolve: resolveJobChain,
					},
					"batch": {
						Type:    BatchType,
						Resolve: resolveJobBatch,
					},
				}
			}),
		})
	}
}

// Define the root query type.
func getQueryType(resolver *Resolver) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"job": {
				Type: JobType,
				Args: graphql.FieldConfigArgument{
					"id": {Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: resolver.ResolveJob,
			},
			"jobs": {
				Type: graphql.NewNonNull(JobConnectionType),
				Args: graphql.FieldConfigArgument{
					"filter": {Type: JobFilterInput},
					"first":  {Type: graphql.Int},
					"after":  {Type: graphql.String},
					"last":   {Type: graphql.Int},
					"before": {Type: graphql.String},
				},
				Resolve: resolver.ResolveJobs,
			},
			"chain": {
				Type: ChainType,
				Args: graphql.FieldConfigArgument{
					"id": {Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: resolver.ResolveChain,
			},
			"chains": {
				Type:    graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(ChainType))),
				Resolve: resolver.ResolveChains,
			},
			"batch": {
				Type: BatchType,
				Args: graphql.FieldConfigArgument{
					"id": {Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: resolver.ResolveBatch,
			},
			"batches": {
				Type:    graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(BatchType))),
				Resolve: resolver.ResolveBatches,
			},
			"crawlState": {
				Type: CrawlStateType,
				Args: graphql.FieldConfigArgument{
					"url": {Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolver.ResolveCrawlState,
			},
			"crawlStates": {
				Type: graphql.NewNonNull(CrawlStateConnectionType),
				Args: graphql.FieldConfigArgument{
					"first":  {Type: graphql.Int},
					"after":  {Type: graphql.String},
					"last":   {Type: graphql.Int},
					"before": {Type: graphql.String},
				},
				Resolve: resolver.ResolveCrawlStates,
			},
			"metrics": {
				Type:    graphql.NewNonNull(MetricsSnapshotType),
				Resolve: resolver.ResolveMetrics,
			},
		},
	})
}

// Define the root mutation type.
func getMutationType(resolver *Resolver) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"createJob": {
				Type: graphql.NewNonNull(JobType),
				Args: graphql.FieldConfigArgument{
					"input": {Type: graphql.NewNonNull(CreateJobInput)},
				},
				Resolve: resolver.ResolveCreateJob,
			},
			"cancelJob": {
				Type: JobType,
				Args: graphql.FieldConfigArgument{
					"id": {Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: resolver.ResolveCancelJob,
			},
			"deleteJob": {
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"id":    {Type: graphql.NewNonNull(graphql.ID)},
					"force": {Type: graphql.Boolean},
				},
				Resolve: resolver.ResolveDeleteJob,
			},
			"createChain": {
				Type: graphql.NewNonNull(ChainType),
				Args: graphql.FieldConfigArgument{
					"input": {Type: graphql.NewNonNull(CreateChainInput)},
				},
				Resolve: resolver.ResolveCreateChain,
			},
			"deleteChain": {
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"id": {Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: resolver.ResolveDeleteChain,
			},
			"deleteBatch": {
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"id": {Type: graphql.NewNonNull(graphql.ID)},
				},
				Resolve: resolver.ResolveDeleteBatch,
			},
			"deleteCrawlState": {
				Type: graphql.NewNonNull(graphql.Boolean),
				Args: graphql.FieldConfigArgument{
					"url": {Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: resolver.ResolveDeleteCrawlState,
			},
		},
	})
}

// Define the root subscription type.
func getSubscriptionType(subscriptionManager *SubscriptionManager) *graphql.Object {
	return graphql.NewObject(graphql.ObjectConfig{
		Name: "Subscription",
		Fields: graphql.Fields{
			"jobStatusChanged": {
				Type: JobType,
				Args: graphql.FieldConfigArgument{
					"jobId": {Type: graphql.ID},
				},
				Resolve:   subscriptionManager.ResolveJobStatusChanged,
				Subscribe: subscriptionManager.SubscribeJobStatusChanged,
			},
			"jobCompleted": {
				Type: JobType,
				Args: graphql.FieldConfigArgument{
					"jobId": {Type: graphql.ID},
				},
				Resolve:   subscriptionManager.ResolveJobCompleted,
				Subscribe: subscriptionManager.SubscribeJobCompleted,
			},
			"metricsUpdated": {
				Type:      MetricsSnapshotType,
				Resolve:   subscriptionManager.ResolveMetricsUpdated,
				Subscribe: subscriptionManager.SubscribeMetricsUpdated,
			},
		},
	})
}

// SchemaConfig holds the configuration for creating a schema.
type SchemaConfig struct {
	Resolver            *Resolver
	SubscriptionManager *SubscriptionManager
}

// NewSchema creates a new GraphQL schema with all types and resolvers.
func NewSchema(config SchemaConfig) (graphql.Schema, error) {
	// Initialize types to avoid initialization cycles
	initTypes()

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:        getQueryType(config.Resolver),
		Mutation:     getMutationType(config.Resolver),
		Subscription: getSubscriptionType(config.SubscriptionManager),
	})
}
