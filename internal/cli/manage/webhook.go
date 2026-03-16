// Package manage provides CLI commands for webhook delivery inspection.
//
// Purpose:
// - Expose persisted webhook delivery history to terminal operators.
//
// Responsibilities:
// - Route `spartan webhook deliveries ...` subcommands.
// - Prefer the local API when available, with direct-store fallback for offline/serverless use.
// - Render sanitized webhook delivery summaries and details.
//
// Scope:
// - Operator-facing delivery inspection only; webhook dispatch and persistence stay in internal/webhook.
//
// Usage:
// - Run `spartan webhook deliveries list` or `spartan webhook deliveries get <id>`.
//
// Invariants/Assumptions:
// - URLs and error text shown to operators must be sanitized.
// - Pagination defaults match the API list endpoint (limit 100, max 1000).
// - Direct-store fallback must work when the API server is not available.
package manage

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

const webhookCommandHelpText = `Inspect persisted webhook deliveries.

Usage:
  spartan webhook deliveries <subcommand> [options]

Subcommands:
  list    List webhook deliveries
  get     Get a single webhook delivery by id

Examples:
  spartan webhook deliveries list
  spartan webhook deliveries list --job-id <job-id> --limit 50 --offset 0
  spartan webhook deliveries get <delivery-id>
`

type webhookDeliveryListResult struct {
	Deliveries []webhook.InspectableDelivery `json:"deliveries"`
	Total      int                           `json:"total"`
	Limit      int                           `json:"limit"`
	Offset     int                           `json:"offset"`
}

type webhookDeliveryListAPIResponse struct {
	Deliveries []webhook.InspectableDelivery `json:"deliveries"`
	Total      int                           `json:"total"`
}

func RunWebhook(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printWebhookHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printWebhookHelp()
		return 0
	}

	switch args[0] {
	case "deliveries":
		return runWebhookDeliveries(ctx, cfg, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown webhook subcommand: %s\n", args[0])
		printWebhookHelp()
		return 1
	}
}

func runWebhookDeliveries(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printWebhookHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printWebhookHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return listWebhookDeliveries(ctx, cfg, args[1:])
	case "get":
		return getWebhookDelivery(ctx, cfg, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown webhook deliveries subcommand: %s\n", args[0])
		printWebhookHelp()
		return 1
	}
}

func listWebhookDeliveries(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("webhook deliveries list", flag.ExitOnError)
	jobID := fs.String("job-id", "", "Filter deliveries by job ID")
	limit := fs.Int("limit", 100, "Maximum number of deliveries to show")
	offset := fs.Int("offset", 0, "Number of deliveries to skip")
	_ = fs.Parse(args)

	normalizedLimit, normalizedOffset, err := normalizeWebhookPage(*limit, *offset)
	if err != nil {
		fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
		return 1
	}

	result, err := fetchWebhookDeliveries(ctx, cfg, strings.TrimSpace(*jobID), normalizedLimit, normalizedOffset)
	if err != nil {
		fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
		return 1
	}

	if len(result.Deliveries) == 0 {
		fmt.Println("No webhook deliveries found.")
		return 0
	}

	fmt.Printf("Webhook deliveries (showing %d of %d, limit=%d, offset=%d):\n", len(result.Deliveries), result.Total, result.Limit, result.Offset)
	fmt.Printf("%-32s %-20s %-18s %-10s %-8s %-6s %-20s\n", "ID", "JOB ID", "EVENT TYPE", "STATUS", "ATTEMPTS", "CODE", "UPDATED")
	fmt.Println(strings.Repeat("-", 130))
	for _, delivery := range result.Deliveries {
		code := "-"
		if delivery.ResponseCode != 0 {
			code = fmt.Sprintf("%d", delivery.ResponseCode)
		}
		fmt.Printf("%-32s %-20s %-18s %-10s %-8d %-6s %-20s\n",
			truncate(delivery.ID, 32),
			truncate(delivery.JobID, 20),
			truncate(string(delivery.EventType), 18),
			string(delivery.Status),
			delivery.Attempts,
			code,
			truncate(delivery.UpdatedAt, 20),
		)
		fmt.Printf("  URL: %s\n", delivery.URL)
		if delivery.LastError != "" {
			fmt.Printf("  Error: %s\n", truncate(delivery.LastError, 120))
		}
	}
	return 0
}

func getWebhookDelivery(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("webhook deliveries get", flag.ExitOnError)
	_ = fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "delivery id is required")
		fmt.Fprintln(os.Stderr, "Usage: spartan webhook deliveries get <delivery-id>")
		return 1
	}

	delivery, err := fetchWebhookDelivery(ctx, cfg, strings.TrimSpace(fs.Arg(0)))
	if err != nil {
		fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
		return 1
	}

	payload, err := json.MarshalIndent(delivery, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, apperrors.SafeMessage(err))
		return 1
	}
	fmt.Println(string(payload))
	return 0
}

func fetchWebhookDeliveries(ctx context.Context, cfg config.Config, jobID string, limit, offset int) (webhookDeliveryListResult, error) {
	var apiErr error
	if isServerRunning(ctx, cfg.Port) {
		result, err := fetchWebhookDeliveriesViaAPI(ctx, cfg.Port, jobID, limit, offset)
		if err == nil {
			return result, nil
		}
		apiErr = err
	}

	result, err := fetchWebhookDeliveriesDirect(ctx, cfg.DataDir, jobID, limit, offset)
	if err == nil {
		return result, nil
	}
	if apiErr != nil {
		return webhookDeliveryListResult{}, apperrors.Wrap(apperrors.KindInternal, "failed to load webhook deliveries", errors.Join(apiErr, err))
	}
	return webhookDeliveryListResult{}, err
}

func fetchWebhookDelivery(ctx context.Context, cfg config.Config, id string) (webhook.InspectableDelivery, error) {
	if strings.TrimSpace(id) == "" {
		return webhook.InspectableDelivery{}, apperrors.Validation("delivery id is required")
	}

	var apiErr error
	if isServerRunning(ctx, cfg.Port) {
		delivery, err := fetchWebhookDeliveryViaAPI(ctx, cfg.Port, id)
		if err == nil {
			return delivery, nil
		}
		apiErr = err
	}

	delivery, err := fetchWebhookDeliveryDirect(ctx, cfg.DataDir, id)
	if err == nil {
		return delivery, nil
	}
	if apiErr != nil && !apperrors.IsKind(err, apperrors.KindNotFound) {
		return webhook.InspectableDelivery{}, apperrors.Wrap(apperrors.KindInternal, "failed to load webhook delivery", errors.Join(apiErr, err))
	}
	return webhook.InspectableDelivery{}, err
}

func fetchWebhookDeliveriesViaAPI(ctx context.Context, port, jobID string, limit, offset int) (webhookDeliveryListResult, error) {
	endpoint := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:" + port,
		Path:   "/v1/webhooks/deliveries",
	}
	query := endpoint.Query()
	if jobID != "" {
		query.Set("job_id", jobID)
	}
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("offset", fmt.Sprintf("%d", offset))
	endpoint.RawQuery = query.Encode()

	resp, err := doWebhookInspectionRequest(ctx, endpoint.String())
	if err != nil {
		return webhookDeliveryListResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return webhookDeliveryListResult{}, decodeWebhookAPIError(resp)
	}

	var payload webhookDeliveryListAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return webhookDeliveryListResult{}, apperrors.Wrap(apperrors.KindInternal, "failed to decode webhook delivery list", err)
	}

	return webhookDeliveryListResult{
		Deliveries: sanitizeInspectableDeliveries(payload.Deliveries),
		Total:      payload.Total,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func fetchWebhookDeliveryViaAPI(ctx context.Context, port, id string) (webhook.InspectableDelivery, error) {
	endpoint := url.URL{
		Scheme: "http",
		Host:   "127.0.0.1:" + port,
		Path:   "/v1/webhooks/deliveries/" + url.PathEscape(id),
	}

	resp, err := doWebhookInspectionRequest(ctx, endpoint.String())
	if err != nil {
		return webhook.InspectableDelivery{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return webhook.InspectableDelivery{}, decodeWebhookAPIError(resp)
	}

	var delivery webhook.InspectableDelivery
	if err := json.NewDecoder(resp.Body).Decode(&delivery); err != nil {
		return webhook.InspectableDelivery{}, apperrors.Wrap(apperrors.KindInternal, "failed to decode webhook delivery", err)
	}
	return webhook.SanitizeInspectableDelivery(delivery), nil
}

func doWebhookInspectionRequest(ctx context.Context, endpoint string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create webhook inspection request", err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query webhook inspection API", err)
	}
	return resp, nil
}

func decodeWebhookAPIError(resp *http.Response) error {
	var payload api.ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil && strings.TrimSpace(payload.Error) != "" {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return apperrors.Validation(payload.Error)
		case http.StatusForbidden, http.StatusUnauthorized:
			return apperrors.Permission(payload.Error)
		case http.StatusNotFound:
			return apperrors.NotFound(payload.Error)
		default:
			return apperrors.Internal(payload.Error)
		}
	}
	if resp.StatusCode == http.StatusNotFound {
		return apperrors.NotFound("webhook delivery not found")
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return apperrors.Permission(fmt.Sprintf("webhook inspection API returned %d", resp.StatusCode))
	}
	return apperrors.Internal(fmt.Sprintf("webhook inspection API returned %d", resp.StatusCode))
}

func fetchWebhookDeliveriesDirect(ctx context.Context, dataDir, jobID string, limit, offset int) (webhookDeliveryListResult, error) {
	deliveryStore, err := openWebhookStore(dataDir)
	if err != nil {
		return webhookDeliveryListResult{}, err
	}

	total, err := deliveryStore.CountRecords(ctx, jobID)
	if err != nil {
		return webhookDeliveryListResult{}, apperrors.Wrap(apperrors.KindInternal, "failed to count webhook deliveries", err)
	}
	records, err := deliveryStore.ListRecords(ctx, jobID, limit, offset)
	if err != nil {
		return webhookDeliveryListResult{}, apperrors.Wrap(apperrors.KindInternal, "failed to list webhook deliveries", err)
	}

	return webhookDeliveryListResult{
		Deliveries: webhook.ToInspectableDeliveries(records),
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

func fetchWebhookDeliveryDirect(ctx context.Context, dataDir, id string) (webhook.InspectableDelivery, error) {
	deliveryStore, err := openWebhookStore(dataDir)
	if err != nil {
		return webhook.InspectableDelivery{}, err
	}
	record, found, err := deliveryStore.GetRecord(ctx, id)
	if err != nil {
		return webhook.InspectableDelivery{}, apperrors.Wrap(apperrors.KindInternal, "failed to get webhook delivery", err)
	}
	if !found {
		return webhook.InspectableDelivery{}, apperrors.NotFound("webhook delivery not found")
	}
	return webhook.ToInspectableDelivery(record), nil
}

func openWebhookStore(dataDir string) (*webhook.Store, error) {
	deliveryStore := webhook.NewStore(dataDir)
	if err := deliveryStore.Load(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to load webhook deliveries", err)
	}
	return deliveryStore, nil
}

func sanitizeInspectableDeliveries(deliveries []webhook.InspectableDelivery) []webhook.InspectableDelivery {
	result := make([]webhook.InspectableDelivery, 0, len(deliveries))
	for _, delivery := range deliveries {
		result = append(result, webhook.SanitizeInspectableDelivery(delivery))
	}
	return result
}

func normalizeWebhookPage(limit, offset int) (int, int, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		return 0, 0, apperrors.Validation("offset must be greater than or equal to 0")
	}
	return limit, offset, nil
}

func printWebhookHelp() {
	fmt.Fprint(os.Stderr, webhookCommandHelpText)
}
