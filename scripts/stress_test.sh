#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR_DEFAULT="${ROOT_DIR}/.data"
OUT_DIR_DEFAULT="${ROOT_DIR}/out/stress"

OPENAI_DOCS="https://platform.openai.com/docs"
TEST_TARGETS_DEFAULT=(
  "https://example.com"
  "https://httpbin.org/html"
  "https://quotes.toscrape.com"
  "https://books.toscrape.com"
)

usage() {
  cat <<'USAGE'
Stress-test Spartan Scraper (real targets, no mocks).

Usage:
  scripts/stress_test.sh [options]

Options:
  --data-dir <path>          Data directory (default: .data)
  --out-dir <path>           Output directory (default: out/stress)
  --openai-docs              Include https://platform.openai.com/docs
  --targets <csv>            Comma-separated extra targets
  --use-playwright           Use Playwright for headless runs
  --headless                 Force headless for scrape/crawl/research
  --timeout <seconds>        Request timeout (default: 30)
  --wait-timeout <seconds>   Max wait time for jobs (default: 600)
  --concurrency <n>           Worker concurrency (default: 6)
  --rate-limit-qps <n>        Per-host rate limit QPS (default: 5)
  --rate-limit-burst <n>      Per-host burst (default: 8)
  --max-pages <n>             Max pages for crawl (default: 60)
  --max-depth <n>             Max depth for crawl (default: 2)
  --research-query <text>     Research query (default: "pricing")
  --skip-mcp                  Skip MCP tool checks
  --skip-scheduler            Skip scheduler checks
  --skip-web                  Skip web build smoke check
  --help                      Show help

Examples:
  scripts/stress_test.sh --openai-docs --use-playwright --headless
  scripts/stress_test.sh --targets https://news.ycombinator.com,https://example.com

Notes:
  - Uses real targets only. No mocks.
  - Uses CLI + API + MCP + scheduler + exporter.

Prerequisites:
  - go (1.25+)
  - pnpm
  - node
  - curl
  - sed
USAGE
}

check_prereqs() {
  local missing=()
  for cmd in go pnpm node curl sed; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      missing+=("$cmd")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "Error: Missing required tools: ${missing[*]}"
    echo "Please install them before running this script."
    exit 1
  fi
}

parse_json_field() {
  local field="$1"
  if command -v jq >/dev/null 2>&1; then
    # Use jq if available (safer, standard)
    # The // "" ensures we output an empty string instead of "null" if the field is missing
    jq -r ".$field // \"\""
  else
    # Fallback to node (part of project dev deps)
    node -e "const fs=require('fs'); try { const d=JSON.parse(fs.readFileSync(0, 'utf-8')); console.log(d['$field']||''); } catch(e){}"
  fi
}

DATA_DIR="$DATA_DIR_DEFAULT"
OUT_DIR="$OUT_DIR_DEFAULT"
INCLUDE_OPENAI=0
EXTRA_TARGETS=()
USE_PLAYWRIGHT=0
FORCE_HEADLESS=0
TIMEOUT_SECS=30
WAIT_TIMEOUT_SECS=600
CONCURRENCY=6
RATE_LIMIT_QPS=5
RATE_LIMIT_BURST=8
MAX_PAGES=60
MAX_DEPTH=2
RESEARCH_QUERY="pricing"
SKIP_MCP=0
SKIP_SCHEDULER=0
SKIP_WEB=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --data-dir) DATA_DIR="$2"; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --openai-docs) INCLUDE_OPENAI=1; shift ;;
    --targets)
      IFS=',' read -r -a EXTRA_TARGETS <<< "$2"
      shift 2
      ;;
    --use-playwright) USE_PLAYWRIGHT=1; shift ;;
    --headless) FORCE_HEADLESS=1; shift ;;
    --timeout) TIMEOUT_SECS="$2"; shift 2 ;;
    --wait-timeout) WAIT_TIMEOUT_SECS="$2"; shift 2 ;;
    --concurrency) CONCURRENCY="$2"; shift 2 ;;
    --rate-limit-qps) RATE_LIMIT_QPS="$2"; shift 2 ;;
    --rate-limit-burst) RATE_LIMIT_BURST="$2"; shift 2 ;;
    --max-pages) MAX_PAGES="$2"; shift 2 ;;
    --max-depth) MAX_DEPTH="$2"; shift 2 ;;
    --research-query) RESEARCH_QUERY="$2"; shift 2 ;;
    --skip-mcp) SKIP_MCP=1; shift ;;
    --skip-scheduler) SKIP_SCHEDULER=1; shift ;;
    --skip-web) SKIP_WEB=1; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "Unknown arg: $1"; usage; exit 1 ;;
  esac
 done

check_prereqs

mkdir -p "$OUT_DIR"

TARGETS=("${TEST_TARGETS_DEFAULT[@]}")
if [[ "$INCLUDE_OPENAI" == "1" ]]; then
  TARGETS+=("$OPENAI_DOCS")
fi
if [[ ${#EXTRA_TARGETS[@]} -gt 0 ]]; then
  for t in "${EXTRA_TARGETS[@]}"; do
    if [[ -n "$t" ]]; then
      TARGETS+=("$t")
    fi
  done
fi

cd "$ROOT_DIR"

export DATA_DIR
export RATE_LIMIT_QPS
export RATE_LIMIT_BURST
export MAX_CONCURRENCY="$CONCURRENCY"
export REQUEST_TIMEOUT_SECONDS="$TIMEOUT_SECS"
export USE_PLAYWRIGHT="$USE_PLAYWRIGHT"

make build >/dev/null

HEADLESS_JSON="false"
if [[ "$FORCE_HEADLESS" == "1" ]]; then
  HEADLESS_JSON="true"
fi
PLAYWRIGHT_JSON="false"
if [[ "$USE_PLAYWRIGHT" == "1" ]]; then
  PLAYWRIGHT_JSON="true"
fi

PLAYWRIGHT_FLAG=""
if [[ "$USE_PLAYWRIGHT" == "1" ]]; then
  PLAYWRIGHT_FLAG="--playwright"
fi
HEADLESS_FLAG=""
if [[ "$FORCE_HEADLESS" == "1" ]]; then
  HEADLESS_FLAG="--headless"
fi

LOG_DIR="$OUT_DIR/logs"
mkdir -p "$LOG_DIR"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

wait_job_api() {
  local job_id="$1"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECS))
  while true; do
    local status
    status=$(curl -fsS "http://127.0.0.1:8741/v1/jobs/${job_id}" | parse_json_field "status")
    if [[ "$status" == "succeeded" ]]; then
      return 0
    fi
    if [[ "$status" == "failed" ]]; then
      return 1
    fi
    if [[ "$WAIT_TIMEOUT_SECS" != "0" && "$SECONDS" -ge "$deadline" ]]; then
      return 1
    fi
    sleep 1
  done
}

./bin/spartan server >"$LOG_DIR/server.log" 2>&1 &
SERVER_PID=$!

for _ in {1..20}; do
  if curl -fsS "http://127.0.0.1:8741/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.5
done

for target in "${TARGETS[@]}"; do
  ./bin/spartan scrape --url "$target" $HEADLESS_FLAG $PLAYWRIGHT_FLAG --wait --wait-timeout "$WAIT_TIMEOUT_SECS" --timeout "$TIMEOUT_SECS" --out "$OUT_DIR/scrape-$(echo "$target" | sed 's#https\?://##;s#[/:]#_#g').json" >/dev/null
 done

for target in "${TARGETS[@]}"; do
  ./bin/spartan crawl --url "$target" $HEADLESS_FLAG $PLAYWRIGHT_FLAG --max-depth "$MAX_DEPTH" --max-pages "$MAX_PAGES" --wait --wait-timeout "$WAIT_TIMEOUT_SECS" --timeout "$TIMEOUT_SECS" --out "$OUT_DIR/crawl-$(echo "$target" | sed 's#https\?://##;s#[/:]#_#g').jsonl" >/dev/null
 done

RESEARCH_URLS=$(IFS=,; echo "${TARGETS[*]}")
./bin/spartan research --query "$RESEARCH_QUERY" --urls "$RESEARCH_URLS" $HEADLESS_FLAG $PLAYWRIGHT_FLAG --wait --wait-timeout "$WAIT_TIMEOUT_SECS" --timeout "$TIMEOUT_SECS" --out "$OUT_DIR/research.jsonl" >/dev/null

SCRAPE_JOB=$(curl -fsS -X POST "http://127.0.0.1:8741/v1/scrape" -H "Content-Type: application/json" -d "{\"url\":\"${TARGETS[0]}\",\"headless\":${HEADLESS_JSON},\"playwright\":${PLAYWRIGHT_JSON},\"timeoutSeconds\":${TIMEOUT_SECS}}")
SCRAPE_JOB_ID=$(echo "$SCRAPE_JOB" | parse_json_field "id")
wait_job_api "$SCRAPE_JOB_ID"
curl -fsS "http://127.0.0.1:8741/v1/jobs/${SCRAPE_JOB_ID}" >"$OUT_DIR/api-job.json"
curl -fsS "http://127.0.0.1:8741/v1/jobs/${SCRAPE_JOB_ID}/results" >"$OUT_DIR/api-results.json"

CRAWL_JOB=$(curl -fsS -X POST "http://127.0.0.1:8741/v1/crawl" -H "Content-Type: application/json" -d "{\"url\":\"${TARGETS[0]}\",\"maxDepth\":${MAX_DEPTH},\"maxPages\":${MAX_PAGES},\"headless\":${HEADLESS_JSON},\"playwright\":${PLAYWRIGHT_JSON},\"timeoutSeconds\":${TIMEOUT_SECS}}")
CRAWL_JOB_ID=$(echo "$CRAWL_JOB" | parse_json_field "id")
wait_job_api "$CRAWL_JOB_ID"
curl -fsS "http://127.0.0.1:8741/v1/jobs/${CRAWL_JOB_ID}" >"$OUT_DIR/api-crawl-job.json"
curl -fsS "http://127.0.0.1:8741/v1/jobs/${CRAWL_JOB_ID}/results" >"$OUT_DIR/api-crawl-results.json"

RESEARCH_JOB=$(curl -fsS -X POST "http://127.0.0.1:8741/v1/research" -H "Content-Type: application/json" -d "{\"query\":\"${RESEARCH_QUERY}\",\"urls\":[\"${TARGETS[0]}\",\"${TARGETS[1]}\"] ,\"maxDepth\":${MAX_DEPTH},\"maxPages\":${MAX_PAGES},\"headless\":${HEADLESS_JSON},\"playwright\":${PLAYWRIGHT_JSON},\"timeoutSeconds\":${TIMEOUT_SECS}}")
RESEARCH_JOB_ID=$(echo "$RESEARCH_JOB" | parse_json_field "id")
wait_job_api "$RESEARCH_JOB_ID"
curl -fsS "http://127.0.0.1:8741/v1/jobs/${RESEARCH_JOB_ID}" >"$OUT_DIR/api-research-job.json"
curl -fsS "http://127.0.0.1:8741/v1/jobs/${RESEARCH_JOB_ID}/results" >"$OUT_DIR/api-research-results.json"

LATEST_JOB_ID=$(./bin/spartan export --job-id $(ls -t "$DATA_DIR/jobs" | head -n 1) --format md --out "$OUT_DIR/export-latest.md" | tail -n 1 || true)

if [[ "$SKIP_SCHEDULER" == "0" ]]; then
  ./bin/spartan schedule add --kind scrape --interval 5 --url "${TARGETS[0]}" --timeout "$TIMEOUT_SECS" >/dev/null
  sleep 7
  ./bin/spartan schedule list >"$OUT_DIR/schedules.txt"
fi

if [[ "$SKIP_MCP" == "0" ]]; then
  printf '{"id":1,"method":"initialize"}\n' | ./bin/spartan mcp >/dev/null
  printf '{"id":2,"method":"tools/list"}\n' | ./bin/spartan mcp >"$OUT_DIR/mcp-tools.json"
  printf '{"id":3,"method":"tools/call","params":{"name":"scrape_page","arguments":{"url":"%s","headless":false}}}\n' "${TARGETS[0]}" | ./bin/spartan mcp >"$OUT_DIR/mcp-scrape.json"
fi

if [[ "$SKIP_WEB" == "0" ]]; then
  (cd web && pnpm run build) >/dev/null
fi

echo "Stress test completed. Outputs in $OUT_DIR"
