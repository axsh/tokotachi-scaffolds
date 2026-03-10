#!/bin/bash
set -euo pipefail

# ============================================================
# release.sh — Full Build, Catalog Generation & Push
#
# 1. Runs build.sh (full build & unit tests)
# 2. Runs templatizer to regenerate catalog data
# 3. Commits and pushes to main branch
#
# Usage:
#   ./scripts/process/release.sh [OPTIONS]
#
# Options:
#   --help    Show this help message
#
# Exit Codes:
#   0 = Release completed successfully
#   1 = Failure at any step
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# --- Helpers ---
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[PASS]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail()    { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }
step()    { echo -e "${CYAN}${BOLD}===> $*${NC}"; }

show_help() {
    cat << 'EOF'
Usage: ./scripts/process/release.sh [OPTIONS]

1. Runs build.sh (full build & unit tests)
2. Runs templatizer to regenerate catalog data
3. Commits and pushes to main branch

Options:
  --help    Show this help message

Exit Codes:
  0 = Release completed successfully
  1 = Failure at any step

EOF
}

# --- Argument Parsing ---
while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            fail "Unknown option: $1"
            ;;
    esac
done

# ============================================================
# Main
# ============================================================
main() {
    echo ""
    echo -e "${BOLD}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}║     Release Pipeline                     ║${NC}"
    echo -e "${BOLD}╚══════════════════════════════════════════╝${NC}"
    echo ""

    local start_time=$SECONDS

    cd "$PROJECT_ROOT"

    # --- Step 1: Full Build ---
    step "Step 1/3: Full Build"
    if "$SCRIPT_DIR/build.sh"; then
        success "Full build completed."
    else
        fail "Full build failed. Aborting release."
    fi

    # --- Step 2: Regenerate Catalog ---
    step "Step 2/3: Regenerate Catalog Data"
    info "Running templatizer..."
    if ./bin/templatizer ./catalog/; then
        success "Catalog data regenerated."
    else
        fail "Templatizer failed. Aborting release."
    fi

    # --- Step 3: Git Commit & Push ---
    step "Step 3/3: Commit & Push to main"

    git add -A
    if git diff --cached --quiet; then
        warn "No changes to commit. Skipping push."
    else
        info "Committing changes..."
        git commit -m "update catalog"
        info "Pushing to main..."
        git push origin main
        success "Pushed to main."
    fi

    local elapsed=$(( SECONDS - start_time ))
    echo ""
    echo -e "${BOLD}─────────────────────────────────────────────${NC}"
    success "Release pipeline completed (${elapsed}s)"
}

main
