#!/usr/bin/env bash
#
# Purpose:
#   - Exercise deterministic CLI, API, MCP, scheduler, exporter, and web flows end-to-end.
#
# Responsibilities:
#   - Start the product server and the local fixture server.
#   - Run scrape/crawl/research flows through CLI and API.
#   - Optionally run a live-network smoke profile.
#
# Scope:
#   - Heavy local validation only; this script is not a deploy tool.
#
# Usage:
#   - scripts/stress_test.sh
#   - scripts/stress_test.sh --network --targets https://example.com
#
# Invariants/Assumptions:
#   - Default mode is deterministic and uses only loopback fixture targets.
#   - Live-network validation is opt-in via --network.
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR_DEFAULT="${ROOT_DIR}/out/stress"
WORK_DIR_DEFAULT="$(mktemp -d "${TMPDIR:-/tmp}/spartan-stress.XXXXXX")"
SPARTAN_BIN="${ROOT_DIR}/bin/spartan"
FIXTURE_HOST_DEFAULT="127.0.0.1"
SERVER_HOST_DEFAULT="127.0.0.1"

OPENAI_DOCS="https://platform.openai.com/docs"
NETWORK_TARGETS_DEFAULT=(
  "https://example.com"
  "https://httpbin.org/html"
  "https://quotes.toscrape.com"
  "https://books.toscrape.com"
)

usage() {
  cat <<'USAGE'
Stress-test Spartan Scraper.

Usage:
  scripts/stress_test.sh [options]

Options:
  --data-dir <path>          Data directory (default: <work-dir>/data)
  --out-dir <path>           Output directory (default: out/stress)
  --work-dir <path>          Command working directory (default: temp dir)
  --fixture-addr <host:port> Local fixture listen address (default: ephemeral 127.0.0.1 port)
  --network                  Use live Internet targets instead of the local fixture
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
  scripts/stress_test.sh
  scripts/stress_test.sh --network --targets https://news.ycombinator.com,https://example.com
  scripts/stress_test.sh --targets https://news.ycombinator.com,https://example.com

Notes:
  - Default mode uses a deterministic local fixture.
  - --network opts into live Internet smoke validation.
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

  echo "Checking browser availability..."
  # Force headless for the check to ensure we actually test browser presence.
  if ! run_spartan scrape --url "$BROWSER_CHECK_URL" --headless $PLAYWRIGHT_FLAG --timeout 10 >/dev/null 2>&1; then
     echo "Warning: Headless browser check failed. Tests might fail if they require a browser."
     echo "Run with --headless --use-playwright to test specific configurations."
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
    node -e "const fs=require('fs'); try { let value=JSON.parse(fs.readFileSync(0, 'utf-8')); for (const part of '$field'.split('.')) { value = value == null ? '' : value[part]; } console.log(value ?? ''); } catch(e){}"
  fi
}

DATA_DIR=""
DATA_DIR_EXPLICIT=0
OUT_DIR="$OUT_DIR_DEFAULT"
WORK_DIR="$WORK_DIR_DEFAULT"
FIXTURE_ADDR=""
FIXTURE_ADDR_EXPLICIT=0
INCLUDE_OPENAI=0
EXTRA_TARGETS=()
USE_PLAYWRIGHT=0
FORCE_HEADLESS=0
USE_NETWORK_TARGETS=0
SERVER_PORT=""
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
    --data-dir) DATA_DIR="$2"; DATA_DIR_EXPLICIT=1; shift 2 ;;
    --out-dir) OUT_DIR="$2"; shift 2 ;;
    --work-dir) WORK_DIR="$2"; shift 2 ;;
    --fixture-addr) FIXTURE_ADDR="$2"; FIXTURE_ADDR_EXPLICIT=1; shift 2 ;;
    --network) USE_NETWORK_TARGETS=1; shift ;;
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

# Need flags for check_prereqs
HEADLESS_FLAG=""
if [[ "$FORCE_HEADLESS" == "1" ]]; then
  HEADLESS_FLAG="--headless"
fi
PLAYWRIGHT_FLAG=""
if [[ "$USE_PLAYWRIGHT" == "1" ]]; then
  PLAYWRIGHT_FLAG="--playwright"
fi

# Build first so spartan binary exists for check_prereqs
make build >/dev/null

run_spartan_in_dir() {
  local run_dir="$1"
  shift
  (
    cd "$run_dir"
    "$SPARTAN_BIN" "$@"
  )
}

run_spartan() {
  run_spartan_in_dir "$WORK_DIR" "$@"
}

reserve_fixture_addr() {
  local host="$1"
  node -e '
    const host = process.argv[1];
    const net = require("net");
    const server = net.createServer();
    server.listen(0, host, () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        console.error("failed to reserve fixture port");
        process.exit(1);
      }
      console.log(`${host}:${address.port}`);
      server.close();
    });
    server.on("error", (error) => {
      console.error(error.message);
      process.exit(1);
    });
  ' "$host"
}

mkdir -p "$OUT_DIR"
mkdir -p "$WORK_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd -P)"
WORK_DIR_DEFAULT="$(cd "$WORK_DIR_DEFAULT" && pwd -P)"
WORK_DIR="$(cd "$WORK_DIR" && pwd -P)"
if [[ "$DATA_DIR_EXPLICIT" == "0" ]]; then
  DATA_DIR="${WORK_DIR}/data"
elif [[ "$DATA_DIR" != /* ]]; then
  DATA_DIR="${WORK_DIR}/${DATA_DIR}"
fi
mkdir -p "$DATA_DIR"
DATA_DIR="$(cd "$DATA_DIR" && pwd -P)"

export DATA_DIR
if [[ -z "$SERVER_PORT" ]]; then
  SERVER_PORT="$(reserve_fixture_addr "$SERVER_HOST_DEFAULT" | awk -F: '{print $NF}')"
fi
export PORT="$SERVER_PORT"
export RATE_LIMIT_QPS
export RATE_LIMIT_BURST
export MAX_CONCURRENCY="$CONCURRENCY"
export REQUEST_TIMEOUT_SECONDS="$TIMEOUT_SECS"
export USE_PLAYWRIGHT="$USE_PLAYWRIGHT"

HEADLESS_JSON="false"
if [[ "$FORCE_HEADLESS" == "1" ]]; then
  HEADLESS_JSON="true"
fi
PLAYWRIGHT_JSON="false"
if [[ "$USE_PLAYWRIGHT" == "1" ]]; then
  PLAYWRIGHT_JSON="true"
fi

LOG_DIR="$OUT_DIR/logs"
mkdir -p "$LOG_DIR"

FIXTURE_PID=""
SERVER_PID=""
CLEANUP_RAN=0

collect_descendant_pids() {
  local pid="$1"
  local child=""

  while read -r child; do
    [[ -z "$child" ]] && continue
    collect_descendant_pids "$child"
    echo "$child"
  done < <(pgrep -P "$pid" 2>/dev/null || true)
}

stop_process_tree() {
  local root_pid="$1"
  local pid=""
  local -a descendants=()

  [[ -z "$root_pid" ]] && return 0
  if ! kill -0 "$root_pid" >/dev/null 2>&1; then
    return 0
  fi

  mapfile -t descendants < <(collect_descendant_pids "$root_pid")

  for pid in "${descendants[@]}"; do
    kill -TERM "$pid" >/dev/null 2>&1 || true
  done
  kill -TERM "$root_pid" >/dev/null 2>&1 || true
  sleep 1

  for pid in "${descendants[@]}"; do
    kill -KILL "$pid" >/dev/null 2>&1 || true
  done
  kill -KILL "$root_pid" >/dev/null 2>&1 || true
  wait "$root_pid" >/dev/null 2>&1 || true
}

cleanup() {
  local exit_code="${1:-$?}"

  if [[ "$CLEANUP_RAN" == "1" ]]; then
    return 0
  fi
  CLEANUP_RAN=1

  echo "Cleaning up..."
  stop_process_tree "${FIXTURE_PID:-}"
  stop_process_tree "${SERVER_PID:-}"
  if [[ -n "${WORK_DIR:-}" && "$WORK_DIR" == "${WORK_DIR_DEFAULT}" ]]; then
    rm -rf "$WORK_DIR"
  fi

  return "$exit_code"
}

handle_signal() {
  local signal_name="$1"
  echo "Received ${signal_name}; shutting down..."
  cleanup 1
  exit 1
}

trap 'cleanup $?' EXIT
trap 'handle_signal INT' INT
trap 'handle_signal TERM' TERM
trap 'handle_signal HUP' HUP

wait_job_api() {
  local job_id="$1"
  local description="${2:-job $job_id}"
  local deadline=$((SECONDS + WAIT_TIMEOUT_SECS))
  echo "Waiting for $description to complete..."
  while true; do
    local output
    output=$(curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${job_id}" 2>&1) || {
      echo "Error fetching job status: $output"
      return 1
    }
    local status
    status=$(echo "$output" | parse_json_field "job.status")
    if [[ "$status" == "succeeded" ]]; then
      echo "$description succeeded."
      return 0
    fi
    if [[ "$status" == "failed" ]]; then
      local err
      err=$(echo "$output" | parse_json_field "job.error")
      echo "$description failed: $err"
      return 1
    fi
    if [[ "$status" == "canceled" ]]; then
      echo "$description was canceled."
      return 1
    fi
    if [[ "$WAIT_TIMEOUT_SECS" != "0" && "$SECONDS" -ge "$deadline" ]]; then
      echo "Timeout waiting for $description after ${WAIT_TIMEOUT_SECS}s"
      return 1
    fi
    sleep 2
  done
}

wait_for_health() {
  local url="$1"
  local label="${2:-service}"
  local deadline=$((SECONDS + 30))
  echo "Waiting for ${label} to become healthy..."
  while true; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      echo "${label} is healthy."
      return 0
    fi
    if [[ "$SECONDS" -ge "$deadline" ]]; then
      echo "Timeout waiting for ${label} health"
      return 1
    fi
    sleep 1
  done
}

if [[ "$USE_NETWORK_TARGETS" == "1" ]]; then
  TARGETS=("${NETWORK_TARGETS_DEFAULT[@]}")
  BROWSER_CHECK_URL="${TARGETS[0]}"
else
  if [[ "$FIXTURE_ADDR_EXPLICIT" == "0" ]]; then
    FIXTURE_ADDR="$(reserve_fixture_addr "$FIXTURE_HOST_DEFAULT")"
  fi
  FIXTURE_BASE_URL="http://${FIXTURE_ADDR}"
  go run ./scripts/serve_testsite.go --addr "$FIXTURE_ADDR" >"$LOG_DIR/fixture.log" 2>&1 &
  FIXTURE_PID=$!
  wait_for_health "${FIXTURE_BASE_URL}/healthz" "fixture" || {
    echo "Fixture failed to start. See $LOG_DIR/fixture.log"
    exit 1
  }
  TARGETS=(
    "${FIXTURE_BASE_URL}/"
    "${FIXTURE_BASE_URL}/html"
    "${FIXTURE_BASE_URL}/research/pricing"
    "${FIXTURE_BASE_URL}/research/faq"
  )
  BROWSER_CHECK_URL="${FIXTURE_BASE_URL}/"
fi

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

check_prereqs

run_spartan server >"$LOG_DIR/server.log" 2>&1 &
SERVER_PID=$!

wait_for_health "http://127.0.0.1:${SERVER_PORT}/healthz" "server" || {
  echo "Server failed to start. See $LOG_DIR/server.log"
  exit 1
}

for target in "${TARGETS[@]}"; do
  run_spartan scrape --url "$target" $HEADLESS_FLAG $PLAYWRIGHT_FLAG --wait --wait-timeout "$WAIT_TIMEOUT_SECS" --timeout "$TIMEOUT_SECS" --out "$OUT_DIR/scrape-$(echo "$target" | sed 's#https\?://##;s#[/:]#_#g').json" >/dev/null
 done

for target in "${TARGETS[@]}"; do
  run_spartan crawl --url "$target" $HEADLESS_FLAG $PLAYWRIGHT_FLAG --max-depth "$MAX_DEPTH" --max-pages "$MAX_PAGES" --wait --wait-timeout "$WAIT_TIMEOUT_SECS" --timeout "$TIMEOUT_SECS" --out "$OUT_DIR/crawl-$(echo "$target" | sed 's#https\?://##;s#[/:]#_#g').jsonl" >/dev/null
 done

RESEARCH_URLS=$(IFS=,; echo "${TARGETS[*]}")
run_spartan research --query "$RESEARCH_QUERY" --urls "$RESEARCH_URLS" $HEADLESS_FLAG $PLAYWRIGHT_FLAG --wait --wait-timeout "$WAIT_TIMEOUT_SECS" --timeout "$TIMEOUT_SECS" --out "$OUT_DIR/research.jsonl" >/dev/null

SCRAPE_JOB=$(curl -fsS -X POST "http://127.0.0.1:${SERVER_PORT}/v1/scrape" -H "Content-Type: application/json" -d "{\"url\":\"${TARGETS[0]}\",\"headless\":${HEADLESS_JSON},\"playwright\":${PLAYWRIGHT_JSON},\"timeoutSeconds\":${TIMEOUT_SECS}}")
SCRAPE_JOB_ID=$(echo "$SCRAPE_JOB" | parse_json_field "job.id")
wait_job_api "$SCRAPE_JOB_ID"
curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${SCRAPE_JOB_ID}" >"$OUT_DIR/api-job.json"
curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${SCRAPE_JOB_ID}/results" >"$OUT_DIR/api-results.json"

CRAWL_JOB=$(curl -fsS -X POST "http://127.0.0.1:${SERVER_PORT}/v1/crawl" -H "Content-Type: application/json" -d "{\"url\":\"${TARGETS[0]}\",\"maxDepth\":${MAX_DEPTH},\"maxPages\":${MAX_PAGES},\"headless\":${HEADLESS_JSON},\"playwright\":${PLAYWRIGHT_JSON},\"timeoutSeconds\":${TIMEOUT_SECS}}")
CRAWL_JOB_ID=$(echo "$CRAWL_JOB" | parse_json_field "job.id")
wait_job_api "$CRAWL_JOB_ID"
curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${CRAWL_JOB_ID}" >"$OUT_DIR/api-crawl-job.json"
curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${CRAWL_JOB_ID}/results" >"$OUT_DIR/api-crawl-results.json"

RESEARCH_JOB=$(curl -fsS -X POST "http://127.0.0.1:${SERVER_PORT}/v1/research" -H "Content-Type: application/json" -d "{\"query\":\"${RESEARCH_QUERY}\",\"urls\":[\"${TARGETS[0]}\",\"${TARGETS[1]}\"] ,\"maxDepth\":${MAX_DEPTH},\"maxPages\":${MAX_PAGES},\"headless\":${HEADLESS_JSON},\"playwright\":${PLAYWRIGHT_JSON},\"timeoutSeconds\":${TIMEOUT_SECS}}")
RESEARCH_JOB_ID=$(echo "$RESEARCH_JOB" | parse_json_field "job.id")
wait_job_api "$RESEARCH_JOB_ID"
curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${RESEARCH_JOB_ID}" >"$OUT_DIR/api-research-job.json"
curl -fsS "http://127.0.0.1:${SERVER_PORT}/v1/jobs/${RESEARCH_JOB_ID}/results" >"$OUT_DIR/api-research-results.json"

# `spartan export --out` enforces writes within the command working directory.
# Run the export from OUT_DIR so the written artifact and persisted destination stay aligned.
run_spartan_in_dir "$OUT_DIR" export --job-id "$RESEARCH_JOB_ID" --format md --out "$OUT_DIR/export-latest.md" >/dev/null

if [[ "$SKIP_SCHEDULER" == "0" ]]; then
  run_spartan schedule add --kind scrape --interval 5 --url "${TARGETS[0]}" --timeout "$TIMEOUT_SECS" >/dev/null
  sleep 7
  run_spartan schedule list >"$OUT_DIR/schedules.txt"
fi

if [[ "$SKIP_MCP" == "0" ]]; then
  printf '{"id":1,"method":"initialize"}\n' | run_spartan mcp >/dev/null
  printf '{"id":2,"method":"tools/list"}\n' | run_spartan mcp >"$OUT_DIR/mcp-tools.json"
  printf '{"id":3,"method":"tools/call","params":{"name":"scrape_page","arguments":{"url":"%s","headless":false}}}\n' "${TARGETS[0]}" | run_spartan mcp >"$OUT_DIR/mcp-scrape.json"
fi

if [[ "$SKIP_WEB" == "0" ]]; then
  (cd "$ROOT_DIR/web" && pnpm run build) >/dev/null
fi

echo "Stress test completed. Outputs in $OUT_DIR"
