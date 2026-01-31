#!/bin/bash
#
# run_ci.sh - Local CI runner for spartan-scraper
#
# RESPONSIBILITIES:
#   - Changes to the repository root (script-relative)
#   - Executes the full CI pipeline via 'make ci'
#
# NON-GOALS:
#   - Does not install dependencies (assumes make install has been run)
#   - Does not handle remote CI or GitHub Actions
#   - Does not modify source code or fix failures automatically
#
# ASSUMPTIONS:
#   - Script is located at the repository root
#   - 'make' and all build tools are available in PATH
#   - Repository has a working 'make ci' target
#
# USAGE:
#   ./run_ci.sh           # Run the full CI pipeline
#   ./run_ci.sh --help    # Show this help message
#
# EXIT CODES:
#   0   CI passed successfully
#   1   CI failed or invalid arguments

set -euo pipefail

# Show help message
show_help() {
    cat << 'EOF'
Usage: ./run_ci.sh [OPTIONS]

Run the local CI pipeline for spartan-scraper.

OPTIONS:
    -h, --help    Show this help message and exit

EXAMPLES:
    # Run the full CI pipeline
    ./run_ci.sh

    # Show help
    ./run_ci.sh --help

This script changes to the repository root and runs 'make ci',
which typically includes: install, generate, format, type-check,
lint, build, and test-ci.

Exit codes:
    0   CI passed successfully
    1   CI failed or invalid arguments
EOF
}

# Parse arguments
if [[ $# -gt 0 ]]; then
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo "Error: Unknown option: $1" >&2
            echo "Use --help for usage information" >&2
            exit 1
            ;;
    esac
fi

# Change to script directory (repo root)
cd "$(dirname "$0")"

# Run CI
make ci 2>&1
