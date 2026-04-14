#!/bin/bash
# =============================================================================
# rebuild-ast.sh — Build or rebuild the sitting_duck AST cache.
#
# Distributed by Esquisse init.sh to project scripts/rebuild-ast.sh.
# Canonical source: duckdb-code/scripts/rebuild-ast.sh (duckdb-skills skill).
# Do not edit this copy — update the skill source and re-copy manually.
#
# Default mode: full rebuild (safe to re-run — uses CREATE OR REPLACE TABLE).
# Incremental mode (--incremental / -i): only re-parses files modified since
# the last cache build, using `find -newer`. Falls back to full rebuild if
# no cache exists yet.
#
# Usage:
#   bash scripts/rebuild-ast.sh [--incremental] [glob_pattern]
#
# Arguments:
#   --incremental / -i   Re-parse only files newer than the existing cache.
#                        Falls back to full rebuild if no cache exists.
#   glob_pattern         File glob to parse (default: auto-detected from project).
#                        Examples:
#                          "**/*.go"
#                          "**/*.py"
#                          "src/**/*.ts"
#
# Environment overrides:
#   AST_DB         Path to the DuckDB cache file (default: code_ast.duckdb)
#   AST_PEEK       Peek character limit (default: 200; use 'full' for complete source)
#
# Examples:
#   bash scripts/rebuild-ast.sh                        # full rebuild, auto-detect
#   bash scripts/rebuild-ast.sh "**/*.go"              # full rebuild, explicit pattern
#   bash scripts/rebuild-ast.sh --incremental          # incremental, auto-detect
#   bash scripts/rebuild-ast.sh --incremental "**/*.go"
#   AST_PEEK=full bash scripts/rebuild-ast.sh "**/*.go"
#   AST_DB=~/.duckdb/code_ast/myproject.duckdb bash scripts/rebuild-ast.sh
# =============================================================================

set -euo pipefail

readonly SCRIPT_NAME="$(basename "$0")"

# ── Helpers ───────────────────────────────────────────────────────────────────
die() { echo "Error: $*" >&2; exit 1; }

# ── Defaults ──────────────────────────────────────────────────────────────────
AST_DB="${AST_DB:-code_ast.duckdb}"
AST_PEEK="${AST_PEEK:-200}"
INCREMENTAL=false
PATTERN=""

# ── Parse args ────────────────────────────────────────────────────────────────
while [[ $# -gt 0 ]]; do
    case "$1" in
        --incremental|-i) INCREMENTAL=true; shift ;;
        -h|--help)
            echo "Usage: $SCRIPT_NAME [--incremental] [glob_pattern]"
            echo ""
            echo "  --incremental, -i   Only re-parse files newer than the existing cache."
            echo "                      Falls back to full rebuild if no cache exists."
            echo "  glob_pattern        File glob (default: auto-detected)"
            echo ""
            echo "Environment: AST_DB (default: code_ast.duckdb), AST_PEEK (default: 200)"
            exit 0 ;;
        --*|-*) die "Unknown flag: $1" ;;
        *) [[ -z "$PATTERN" ]] && PATTERN="$1" || die "Unexpected argument: $1"
           shift ;;
    esac
done

# ── Find DuckDB ───────────────────────────────────────────────────────────────
DUCKDB=""
if command -v duckdb &>/dev/null; then
    DUCKDB="$(command -v duckdb)"
elif [[ -L "$HOME/.duckdb/cli/latest" ]]; then
    DUCKDB="$HOME/.duckdb/cli/latest/duckdb"
else
    LATEST_VER=$(ls -d "$HOME"/.duckdb/cli/[0-9]*/ 2>/dev/null \
        | sed 's|.*/\([0-9][^/]*\)/|\1|' \
        | sort -t. -k1,1n -k2,2n -k3,3n \
        | tail -1)
    [[ -n "$LATEST_VER" ]] && DUCKDB="$HOME/.duckdb/cli/$LATEST_VER/duckdb"
fi
[[ -x "$DUCKDB" ]] || die "duckdb not found. Install from https://duckdb.org/docs/installation/"

# ── Determine glob pattern ────────────────────────────────────────────────────
if [[ -z "$PATTERN" ]]; then
    declare -A COUNTS
    for ext in go py ts tsx js rs cpp c java; do
        COUNTS[$ext]=$(find . -name "*.$ext" \
            -not -path "./.git/*" \
            -not -path "./vendor/*" \
            -not -path "./node_modules/*" \
            -not -path "./gen/*" \
            2>/dev/null | wc -l)
    done

    BEST_EXT=""
    BEST_COUNT=0
    for ext in "${!COUNTS[@]}"; do
        if [[ ${COUNTS[$ext]} -gt $BEST_COUNT ]]; then
            BEST_COUNT=${COUNTS[$ext]}
            BEST_EXT=$ext
        fi
    done

    if [[ -z "$BEST_EXT" || $BEST_COUNT -eq 0 ]]; then
        die "No source files found and no glob_pattern provided.
Usage: $SCRIPT_NAME [--incremental] [glob_pattern]  e.g. \"**/*.go\""
    fi

    PATTERN="**/*.$BEST_EXT"
    echo "Auto-detected: $BEST_COUNT .$BEST_EXT files → using pattern: $PATTERN"
fi

# ── Incremental fallback check ────────────────────────────────────────────────
if [[ "$INCREMENTAL" == true && ! -f "$AST_DB" ]]; then
    echo "No existing cache found at $AST_DB — falling back to full rebuild."
    INCREMENTAL=false
fi

# ── DB directory creation ─────────────────────────────────────────────────────
DB_DIR="$(dirname "$AST_DB")"
[[ -n "$DB_DIR" && "$DB_DIR" != "." ]] && mkdir -p "$DB_DIR"

# =============================================================================
# Incremental mode — re-parse only files newer than the cache
# =============================================================================
if [[ "$INCREMENTAL" == true ]]; then
    # Derive file extension from pattern for find filter.
    # Works for patterns like "**/*.go", "src/**/*.ts", "**/*.py".
    EXT="${PATTERN##*.}"

    # Find source files modified since the cache was last written.
    # Strip leading "./" so paths match the file_path values stored by read_ast.
    TMPFILE=$(mktemp /tmp/ast_changed_XXXXXX.txt)
    trap 'rm -f "$TMPFILE"' EXIT

    find . -name "*.$EXT" -newer "$AST_DB" \
        -not -path "./.git/*" \
        -not -path "./vendor/*" \
        -not -path "./node_modules/*" \
        -not -path "./gen/*" \
        | sed 's|^\./||' \
        | sort \
        > "$TMPFILE"

    CHANGED_COUNT=$(grep -c . "$TMPFILE" 2>/dev/null || echo 0)

    if [[ "$CHANGED_COUNT" -eq 0 ]]; then
        echo "Cache is up to date — no .$EXT files changed since last build."
        echo "  Cache file : $AST_DB"
        exit 0
    fi

    echo "Incremental update: $CHANGED_COUNT changed .$EXT file(s)"
    sed 's/^/  /' "$TMPFILE"
    echo ""
    echo "  Peek : $AST_PEEK"
    echo ""

    # Build a DuckDB array literal from the file list.
    # Single quotes in paths are escaped by doubling them (SQL standard).
    FILES_ARRAY="["
    first=true
    while IFS= read -r filepath; do
        escaped="${filepath//\'/\'\'}"
        if [[ "$first" == true ]]; then
            FILES_ARRAY="${FILES_ARRAY}'${escaped}'"
            first=false
        else
            FILES_ARRAY="${FILES_ARRAY}, '${escaped}'"
        fi
    done < "$TMPFILE"
    FILES_ARRAY="${FILES_ARRAY}]"

    START_SEC=$SECONDS

    # Unquoted heredoc: expands $FILES_ARRAY and $AST_PEEK.
    # Paths are escaped above; no SQL injection risk from project-local paths.
"$DUCKDB" -init /dev/null "$AST_DB" -jsonlines <<SQL
LOAD sitting_duck;
DELETE FROM ast WHERE file_path IN (SELECT unnest(${FILES_ARRAY}));
INSERT INTO ast
    SELECT * FROM read_ast(${FILES_ARRAY}, ignore_errors := true, peek := ${AST_PEEK});
SQL
    EXIT_CODE=$?

    if [[ $EXIT_CODE -ne 0 ]]; then
        echo "Incremental update failed (exit $EXIT_CODE)." >&2
        exit $EXIT_CODE
    fi

    ELAPSED=$(( SECONDS - START_SEC ))
    ROW_COUNT=$("$DUCKDB" -init /dev/null "$AST_DB" -csv -noheader \
        -c "LOAD sitting_duck; SELECT count(*) FROM ast;" 2>/dev/null || echo "?")
    FILE_COUNT=$("$DUCKDB" -init /dev/null "$AST_DB" -csv -noheader \
        -c "LOAD sitting_duck; SELECT count(DISTINCT file_path) FROM ast;" 2>/dev/null || echo "?")

    echo "Done in ${ELAPSED}s — $CHANGED_COUNT file(s) updated."
    echo "  Total files in cache : $FILE_COUNT"
    echo "  Total AST nodes      : $ROW_COUNT"
    echo "  Cache file           : $AST_DB"
    exit 0
fi

# =============================================================================
# Full rebuild
# =============================================================================
echo "Building AST cache: $AST_DB"
echo "  Pattern : $PATTERN"
echo "  Peek    : $AST_PEEK"
echo ""

START_SEC=$SECONDS

"$DUCKDB" -init /dev/null "$AST_DB" -jsonlines <<SQL
LOAD sitting_duck;
CREATE OR REPLACE TABLE ast AS
SELECT * FROM read_ast('${PATTERN}', ignore_errors := true, peek := ${AST_PEEK});
SQL
EXIT_CODE=$?

if [[ $EXIT_CODE -ne 0 ]]; then
    echo "Cache build failed (exit $EXIT_CODE)." >&2
    exit $EXIT_CODE
fi

ELAPSED=$(( SECONDS - START_SEC ))

# ── Report ────────────────────────────────────────────────────────────────────
ROW_COUNT=$("$DUCKDB" -init /dev/null "$AST_DB" -csv -noheader \
    -c "LOAD sitting_duck; SELECT count(*) FROM ast;" 2>/dev/null || echo "?")
FILE_COUNT=$("$DUCKDB" -init /dev/null "$AST_DB" -csv -noheader \
    -c "LOAD sitting_duck; SELECT count(DISTINCT file_path) FROM ast;" 2>/dev/null || echo "?")

echo "Done in ${ELAPSED}s."
echo "  Files parsed : $FILE_COUNT"
echo "  AST nodes    : $ROW_COUNT"
echo "  Cache file   : $AST_DB"
echo ""
echo "Load macros and query:"
echo "  duckdb -init /dev/null $AST_DB <<'SQL'"
echo "  LOAD sitting_duck;"
echo "  .read scripts/macros.sql"
echo "  SELECT * FROM find_definitions('${PATTERN}');"
echo "  SQL"
