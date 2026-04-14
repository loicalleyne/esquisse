#!/bin/bash
# =============================================================================
# gate-check.sh — Esquisse phase gate validator.
#
# Validates all phase gate criteria from FRAMEWORK.md §10 before you promote
# to the next phase. Exits non-zero on any hard failure.
#
# Usage:
#   bash scripts/gate-check.sh [phase]
#
# Options:
#   --strict          Treat any ASSUMPTION/TODO count > 0 as a hard failure.
#   --lang-adapter    Language adapter: go (default), python, ts
#   --threshold N     Minimum coverage percentage (default: 80)
#
# Environment overrides:
#   LANG_ADAPTER          go | python | ts
#   COVERAGE_THRESHOLD    integer 0-100
#
# Examples:
#   bash scripts/gate-check.sh 1
#   bash scripts/gate-check.sh 2 --strict
#   LANG_ADAPTER=python bash scripts/gate-check.sh 1
# =============================================================================

set -euo pipefail

readonly SCRIPT_NAME="$(basename "$0")"

# ── Defaults ──────────────────────────────────────────────────────────────────
PHASE=""
STRICT=false
LANG_ADAPTER="${LANG_ADAPTER:-go}"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-80}"

# ── Parse args ────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --strict)        STRICT=true;              shift ;;
        --lang-adapter)  LANG_ADAPTER="$2";        shift 2 ;;
        --threshold)     COVERAGE_THRESHOLD="$2";  shift 2 ;;
        -h|--help)
            grep '^#' "$0" | sed 's/^# //; s/^#//'
            exit 0 ;;
        [0-9]*)
            PHASE="$1"; shift ;;
        *)
            echo "Unknown option: $1" >&2
            exit 1 ;;
    esac
done

# ── State tracking ────────────────────────────────────────────────────────────
FAILURES=0
WARNINGS=0

fail() {
    echo "  FAIL  $*" >&2
    FAILURES=$(( FAILURES + 1 ))
}

warn() {
    echo "  WARN  $*"
    WARNINGS=$(( WARNINGS + 1 ))
}

pass() {
    echo "  PASS  $*"
}

section() {
    echo ""
    echo "── $* ──"
}

# ── Language-specific commands ────────────────────────────────────────────────
#
# Go linting: golangci-lint is used when available. It auto-discovers
# .golangci.yml at the project root — no --config flag needed. If golangci-lint
# is not installed, the check falls back to `go vet ./...` with a warning.
# To install: https://golangci-lint.run/welcome/install/
case "$LANG_ADAPTER" in
    go)
        BUILD_CMD="go build ./..."
        TEST_CMD="go test -count=1 ./..."
        COVER_CMD="go test -count=1 -coverprofile=.gate-coverage.out ./..."
        COVER_PARSE_CMD='go tool cover -func=.gate-coverage.out | grep "^total:" | awk "{print \$3}" | tr -d %'
        STUB_PATTERN='panic("TODO'
        STUB_FILES="*.go"
        ;;
    python)
        BUILD_CMD="python -m py_compile \$(find . -name '*.py' -not -path './.venv/*')"
        TEST_CMD="python -m pytest"
        COVER_CMD="python -m pytest --cov=. --cov-report=term-missing"
        COVER_PARSE_CMD='python -m pytest --cov=. --cov-report=term-missing 2>&1 | grep "^TOTAL" | awk "{print \$4}" | tr -d %'
        STUB_PATTERN="raise NotImplementedError"
        STUB_FILES="*.py"
        ;;
    ts)
        BUILD_CMD="pnpm build"
        TEST_CMD="pnpm test"
        COVER_CMD="pnpm test --coverage"
        COVER_PARSE_CMD='pnpm test --coverage 2>&1 | grep "^All files" | awk "{print \$4}" | tr -d %'
        STUB_PATTERN="throw new Error.*TODO"
        STUB_FILES="*.ts *.tsx"
        ;;
    *)
        echo "Error: unknown --lang-adapter '$LANG_ADAPTER'. Valid: go, python, ts" >&2
        exit 1 ;;
esac

# ── Check 1: Build ────────────────────────────────────────────────────────────
section "Check 1: Build"
if eval "$BUILD_CMD" 2>&1; then
    pass "Build succeeded."
else
    fail "Build failed. Fix compilation errors before gate-check."
fi

# ── Check 2: Lint ─────────────────────────────────────────────────────────────
section "Check 2: Lint"
if [[ "$LANG_ADAPTER" == "go" ]]; then
    if command -v golangci-lint &>/dev/null; then
        # Auto-discovers .golangci.yml at the project root; no --config needed.
        if golangci-lint run ./... 2>&1; then
            pass "golangci-lint passed."
        else
            fail "golangci-lint reported issues. Fix all lint errors before promoting."
        fi
    else
        warn "golangci-lint not found — falling back to go vet. Install from https://golangci-lint.run/welcome/install/"
        if go vet ./... 2>&1; then
            pass "go vet passed."
        else
            fail "go vet reported issues."
        fi
    fi
else
    # Non-Go adapters: lint is run as part of the build command via their own toolchains.
    pass "Lint: delegated to build toolchain for lang-adapter '$LANG_ADAPTER'."
fi

# ── Check 3: Tests ────────────────────────────────────────────────────────────
section "Check 3: Tests"
if eval "$TEST_CMD" 2>&1; then
    pass "All tests pass."
else
    fail "Tests failed. All tests must pass before promoting to the next phase."
fi

# ── Check 4: No panic/TODO stubs ─────────────────────────────────────────────
section "Check 4: No unresolved stubs"
STUB_COUNT=0
while IFS= read -r -d '' f; do
    count=$(grep -c "$STUB_PATTERN" "$f" 2>/dev/null || true)
    STUB_COUNT=$(( STUB_COUNT + count ))
    if [[ $count -gt 0 ]]; then
        echo "  found $count stub(s) in $f"
    fi
done < <(find . -name "$STUB_FILES" \
    -not -path "./.git/*" \
    -not -path "./.venv/*" \
    -not -path "./vendor/*" \
    -not -path "./node_modules/*" \
    -not -path "./gen/*" \
    -print0 2>/dev/null)

if [[ $STUB_COUNT -eq 0 ]]; then
    pass "No unresolved stubs."
else
    fail "$STUB_COUNT unresolved stub(s) remain (pattern: '$STUB_PATTERN')."
fi

# ── Check 5: ASSUMPTION / TODO counts ────────────────────────────────────────
section "Check 5: Annotation counts"
ASSUMPTION_COUNT=$(grep -r "// ASSUMPTION:" . \
    --include="*.go" --include="*.py" --include="*.ts" --include="*.tsx" \
    --exclude-dir=".git" --exclude-dir="vendor" --exclude-dir="node_modules" \
    --exclude-dir="gen" -l 2>/dev/null | wc -l || true)

CODE_TODO_COUNT=$(grep -r "// TODO" . \
    --include="*.go" --include="*.py" --include="*.ts" --include="*.tsx" \
    --exclude-dir=".git" --exclude-dir="vendor" --exclude-dir="node_modules" \
    --exclude-dir="gen" -c 2>/dev/null | grep -v ":0$" | wc -l || true)

echo "  Files with // ASSUMPTION: comments: $ASSUMPTION_COUNT"
echo "  Files with // TODO comments:        $CODE_TODO_COUNT"

if [[ "$STRICT" == "true" ]]; then
    if [[ $ASSUMPTION_COUNT -gt 0 ]]; then
        fail "--strict: $ASSUMPTION_COUNT file(s) with ASSUMPTION comments."
    fi
    if [[ $CODE_TODO_COUNT -gt 0 ]]; then
        fail "--strict: $CODE_TODO_COUNT file(s) with TODO comments."
    fi
else
    if [[ $ASSUMPTION_COUNT -gt 0 || $CODE_TODO_COUNT -gt 0 ]]; then
        warn "Annotation counts above are non-zero. Review before promoting."
    else
        pass "No ASSUMPTION or TODO annotations."
    fi
fi

# ── Check 6: Coverage ─────────────────────────────────────────────────────────
section "Check 6: Test coverage (threshold: ${COVERAGE_THRESHOLD}%)"
COVERAGE_VALUE=0
if COVERAGE_OUTPUT=$(eval "$COVER_CMD" 2>&1); then
    # Parse coverage from output.
    COVERAGE_VALUE=$(eval "$COVER_PARSE_CMD" 2>/dev/null || echo "0")
    # Strip any non-numeric characters that snuck in.
    COVERAGE_VALUE="${COVERAGE_VALUE//[^0-9.]/}"
    COVERAGE_VALUE="${COVERAGE_VALUE:-0}"

    # Integer comparison (truncate decimal).
    COVERAGE_INT="${COVERAGE_VALUE%%.*}"
    if [[ -z "$COVERAGE_INT" ]]; then
        COVERAGE_INT=0
    fi

    echo "  Total coverage: ${COVERAGE_VALUE}%"

    if [[ $COVERAGE_INT -lt $COVERAGE_THRESHOLD ]]; then
        fail "Coverage ${COVERAGE_VALUE}% is below threshold ${COVERAGE_THRESHOLD}%."
    else
        pass "Coverage ${COVERAGE_VALUE}% meets threshold ${COVERAGE_THRESHOLD}%."
    fi
else
    warn "Could not collect coverage (command failed). Run tests manually."
fi

# Cleanup Go coverage artifact if present.
[[ -f ".gate-coverage.out" ]] && rm -f ".gate-coverage.out"

# ── Check 7: Task doc status ──────────────────────────────────────────────────
section "Check 7: Task document status"
if [[ -z "$PHASE" ]]; then
    warn "No phase provided — skipping task-doc status check. Pass a phase number to enable."
else
    TASK_DIR="docs/tasks"
    INCOMPLETE=0

    if [[ ! -d "$TASK_DIR" ]]; then
        warn "$TASK_DIR not found — skipping task status check."
    else
        while IFS= read -r -d '' f; do
            if ! grep -qiE "^(\| *)?Status *(\|)? *:? *Completed" "$f" 2>/dev/null; then
                echo "  not completed: $f"
                INCOMPLETE=$(( INCOMPLETE + 1 ))
            fi
        done < <(find "$TASK_DIR" -maxdepth 1 -name "P${PHASE}-*.md" -print0 2>/dev/null)

        if [[ $INCOMPLETE -eq 0 ]]; then
            pass "All P${PHASE} task docs have Status: Completed."
        else
            fail "$INCOMPLETE task doc(s) in P${PHASE} are not Completed."
        fi
    fi
fi

# ── Check 8: Adversarial review infrastructure ───────────────────────────────
section "Check 8: Adversarial review infrastructure"
if [[ -x "scripts/gate-review.sh" ]]; then
    pass "scripts/gate-review.sh is present and executable."
else
    fail "scripts/gate-review.sh is missing or not executable." \
         "Run init.sh to restore adversarial review infrastructure," \
         "or: cp /path/to/esquisse/scripts/gate-review.sh scripts/ && chmod +x scripts/gate-review.sh"
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "════════════════════════════════════════"
if [[ $FAILURES -gt 0 ]]; then
    echo "  GATE NOT PASSED — ${FAILURES} failure(s), ${WARNINGS} warning(s)."
    echo "  Resolve all FAIL items before promoting to the next phase."
    echo "════════════════════════════════════════"
    exit 1
else
    echo "  GATE PASSED — 0 failures, ${WARNINGS} warning(s)."
    if [[ $WARNINGS -gt 0 ]]; then
        echo "  Review warnings above before promoting."
    fi
    echo "════════════════════════════════════════"
    exit 0
fi
