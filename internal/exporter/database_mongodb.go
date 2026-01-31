// Package exporter provides MongoDB database export implementation.
//
// This file contains MongoDB-specific export functions including
// connection handling, batch operations with bulk write, and upsert support.
//
// This file does NOT handle:
// - PostgreSQL or MySQL exports (see database_postgres.go, database_mysql.go)
// - Schema definitions (see database_schema.go)
// - SQL query building (see database_helpers.go)
package exporter

import (
	"context"
	"io"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// exportMongoDBStream exports job results to MongoDB.
func exportMongoDBStream(job model.Job, r io.Reader, cfg DatabaseExportConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connStr := resolveConnectionString("mongodb", cfg.ConnectionString)
	if connStr == "" {
		return apperrors.Validation("mongodb connection string required (set SPARTAN_MONGODB_URL or pass connectionString)")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to connect to mongodb", err)
	}
	defer client.Disconnect(ctx)

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to ping mongodb", err)
	}

	dbName := cfg.Database
	if dbName == "" {
		dbName = "spartan"
	}

	collName := cfg.Table
	if collName == "" {
		collName = defaultCollectionName(job)
	}

	coll := client.Database(dbName).Collection(collName)

	switch job.Kind {
	case model.KindScrape:
		return exportScrapeToMongoDB(ctx, coll, r, cfg)
	case model.KindCrawl:
		return exportCrawlToMongoDB(ctx, coll, r, cfg)
	case model.KindResearch:
		return exportResearchToMongoDB(ctx, coll, r, cfg)
	default:
		return apperrors.Internal("unknown job kind")
	}
}

// exportScrapeToMongoDB exports a single scrape result to MongoDB.
func exportScrapeToMongoDB(ctx context.Context, coll *mongo.Collection, r io.Reader, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ScrapeResult](r)
	if err != nil {
		return err
	}

	doc := bson.M{
		"url":          item.URL,
		"status":       item.Status,
		"title":        safe(item.Normalized.Title, item.Title),
		"text":         item.Text,
		"description":  safe(item.Normalized.Description, item.Metadata.Description),
		"extracted_at": time.Now(),
	}

	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		filter := bson.M{cfg.UpsertKey: item.URL}
		update := bson.M{"$set": doc}
		opts := options.UpdateOne().SetUpsert(true)
		_, err = coll.UpdateOne(ctx, filter, update, opts)
	} else {
		_, err = coll.InsertOne(ctx, doc)
	}

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert scrape result to mongodb", err)
	}
	return nil
}

// exportCrawlToMongoDB uses InsertMany for batch document insertion.
func exportCrawlToMongoDB(ctx context.Context, coll *mongo.Collection, r io.Reader, cfg DatabaseExportConfig) error {
	rs, cleanup, err := ensureSeekable(r)
	if err != nil {
		return err
	}
	defer cleanup()

	const batchSize = 100
	var docs []interface{}

	err = scanReader[CrawlResult](rs, func(item CrawlResult) error {
		doc := bson.M{
			"url":          item.URL,
			"status":       item.Status,
			"title":        safe(item.Normalized.Title, item.Title),
			"text":         item.Text,
			"extracted_at": time.Now(),
		}
		docs = append(docs, doc)

		if len(docs) >= batchSize {
			if err := flushMongoDBBatch(ctx, coll, docs, cfg); err != nil {
				return err
			}
			docs = docs[:0]
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Insert remaining
	if len(docs) > 0 {
		return flushMongoDBBatch(ctx, coll, docs, cfg)
	}
	return nil
}

// flushMongoDBBatch executes a batch insert for MongoDB.
func flushMongoDBBatch(ctx context.Context, coll *mongo.Collection, docs []interface{}, cfg DatabaseExportConfig) error {
	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		// Use bulk write for upsert
		var models []mongo.WriteModel
		for _, doc := range docs {
			docMap, ok := doc.(bson.M)
			if !ok {
				continue
			}
			filter := bson.M{cfg.UpsertKey: docMap[cfg.UpsertKey]}
			update := bson.M{"$set": docMap}
			models = append(models, mongo.NewUpdateOneModel().
				SetFilter(filter).
				SetUpdate(update).
				SetUpsert(true))
		}
		if len(models) > 0 {
			opts := options.BulkWrite().SetOrdered(false)
			_, err := coll.BulkWrite(ctx, models, opts)
			if err != nil {
				return apperrors.Wrap(apperrors.KindInternal, "failed to bulk write mongodb documents", err)
			}
		}
	} else {
		opts := options.InsertMany().SetOrdered(false)
		_, err := coll.InsertMany(ctx, docs, opts)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to insert mongodb documents", err)
		}
	}
	return nil
}

// exportResearchToMongoDB exports research results to MongoDB.
func exportResearchToMongoDB(ctx context.Context, coll *mongo.Collection, r io.Reader, cfg DatabaseExportConfig) error {
	item, err := parseSingleReader[ResearchResult](r)
	if err != nil {
		return err
	}

	// Convert evidence to BSON
	var evidence []bson.M
	for _, ev := range item.Evidence {
		evidence = append(evidence, bson.M{
			"url":          ev.URL,
			"title":        ev.Title,
			"snippet":      ev.Snippet,
			"score":        ev.Score,
			"confidence":   ev.Confidence,
			"cluster_id":   ev.ClusterID,
			"citation_url": ev.CitationURL,
		})
	}

	doc := bson.M{
		"query":        item.Query,
		"summary":      item.Summary,
		"confidence":   item.Confidence,
		"evidence":     evidence,
		"extracted_at": time.Now(),
	}

	if cfg.Mode == "upsert" && cfg.UpsertKey != "" {
		filter := bson.M{cfg.UpsertKey: item.Query}
		update := bson.M{"$set": doc}
		opts := options.UpdateOne().SetUpsert(true)
		_, err = coll.UpdateOne(ctx, filter, update, opts)
	} else {
		_, err = coll.InsertOne(ctx, doc)
	}

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to insert research result to mongodb", err)
	}
	return nil
}
