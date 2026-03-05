#!/bin/bash
#
# run_ci.sh - Local CI profile runner for spartan-scraper
#
# PURPOSE:
#   Provide a single, documented wrapper for running local CI profiles.
#
# RESPONSIBILITIES:
#   - Changes to the repository root (script-relative)
#   - Runs a selected Makefile CI target (`ci-pr`, `ci`, `ci-slow`, `ci-network`, `ci-manual`)
#   - Provides a clear help menu with examples and exit codes
#
# NON-GOALS:
#   - Does not implement remote CI/GitHub Actions
#   - Does not auto-fix failures
#
# ASSUMPTIONS:
#   - Script is located at repository root
#   - `make` and required toolchain are available in PATH
#   - Makefile defines CI profile targets
#
# EXIT CODES:
#   0   CI profile passed successfully
#   1   Runtime failure while executing selected CI profile
#   2   Usage/validation error

set -euo pipefail

DEFAULT_PROFILE="full"

show_help() {
    cat << 'EOF'
Usage: ./run_ci.sh [OPTIONS]

Run a local CI profile for spartan-scraper.

Options:
  -h, --help              Show this help message and exit
  --profile <name>        CI profile to run (default: full)
                          Values:
                            pr      -> make ci-pr
                            full    -> make ci
                            slow    -> make ci-slow
                            network -> make ci-network
                            manual  -> make ci-manual

Examples:
  # Full local CI gate
  ./run_ci.sh

  # Fast PR-equivalent gate (requires clean git state)
  ./run_ci.sh --profile pr

  # Deterministic heavy stress/e2e checks
  ./run_ci.sh --profile slow

  # Optional live-Internet smoke validation
  ./run_ci.sh --profile network

Exit codes:
  0   CI profile passed
  1   Runtime failure during CI execution
  2   Usage or validation error
EOF
}

PROFILE="$DEFAULT_PROFILE"

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        --profile)
            shift
            if [[ $# -eq 0 ]]; then
                echo "Error: --profile requires a value" >&2
                echo "Use --help for usage information" >&2
                exit 2
            fi
            PROFILE="$1"
            ;;
        --profile=*)
            PROFILE="${1#*=}"
            ;;
        *)
            echo "Error: Unknown option: $1" >&2
            echo "Use --help for usage information" >&2
            exit 2
            ;;
    esac
    shift
done

case "$PROFILE" in
    pr)
        TARGET="ci-pr"
        ;;
    full)
        TARGET="ci"
        ;;
    slow)
        TARGET="ci-slow"
        ;;
    network)
        TARGET="ci-network"
        ;;
    manual)
        TARGET="ci-manual"
        ;;
    *)
        echo "Error: Invalid profile '$PROFILE'" >&2
        echo "Valid profiles: pr, full, slow, network, manual" >&2
        exit 2
        ;;
esac

cd "$(dirname "$0")"
make "$TARGET" 2>&1
