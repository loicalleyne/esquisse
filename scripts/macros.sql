-- =============================================================================
-- macros.sql — Esquisse universal AST analysis macros.
--
-- Reusable DuckDB TABLE macros for codebase analysis via sitting_duck.
-- Works across all 27 languages supported by sitting_duck (Go, Python,
-- TypeScript, Rust, C/C++, Java, C#, Ruby, PHP, Swift, Kotlin, and more).
--
-- Prerequisites:
--   LOAD sitting_duck;
--
-- Usage (load into a session):
--   duckdb -init /dev/null code_ast.duckdb <<'SQL'
--   LOAD sitting_duck;
--   .read scripts/macros.sql
--   SELECT * FROM find_definitions('**/*.go', 'Transcoder%');
--   SQL
--
-- Macro index:
--   find_definitions(file_pattern, name_pattern)
--   find_calls(file_pattern, name_pattern)
--   find_imports(file_pattern)
--   find_in_ast(file_pattern, kind, name_pattern)
--   code_structure(file_pattern)
--   complexity_hotspots(file_pattern, n)
--   function_callers(file_pattern, func_name)
--   module_dependencies(file_pattern, package_prefix)
--
-- Optional (require duck_tails extension — git integration):
--   structural_diff(file, from_rev, to_rev, repo)
--   changed_function_summary(from_rev, to_rev, file_pattern, repo)
--
-- Adapted from github.com/teaguesterling/fledgling/sql/code.sql and
-- structural.sql (MIT license). Simplified for Esquisse framework use.
-- =============================================================================

-- find_definitions: Find function, class, or variable definitions.
-- Answers "where is X defined?" — replaces grep for structural search.
--
-- Default (no name_pattern): only top-level function/class/module definitions.
-- With name_pattern: also includes variable definitions at any depth.
--
-- Examples:
--   SELECT * FROM find_definitions('**/*.go');
--   SELECT * FROM find_definitions('**/*.go', 'Transcoder%');
CREATE OR REPLACE MACRO find_definitions(file_pattern, name_pattern := '%') AS TABLE
    SELECT
        file_path,
        name,
        semantic_type_to_string(semantic_type) AS kind,
        start_line,
        end_line,
        peek AS signature
    FROM read_ast(file_pattern, ignore_errors := true)
    WHERE name != ''
      AND name LIKE name_pattern
      AND (
          (name_pattern = '%'
              AND (is_function_definition(semantic_type)
                   OR is_class_definition(semantic_type)
                   OR is_module_definition(semantic_type))
              AND depth <= 2)
          OR
          (name_pattern != '%'
              AND is_definition(semantic_type))
      )
    ORDER BY file_path, start_line;

-- find_calls: Find function/method call sites.
-- Answers "where is this function called?"
--
-- Examples:
--   SELECT * FROM find_calls('**/*.go');
--   SELECT * FROM find_calls('internal/**/*.go', 'AppendRaw%');
CREATE OR REPLACE MACRO find_calls(file_pattern, name_pattern := '%') AS TABLE
    SELECT
        file_path,
        name,
        start_line,
        peek AS call_expression
    FROM read_ast(file_pattern, ignore_errors := true)
    WHERE is_call(semantic_type)
      AND name LIKE name_pattern
    ORDER BY file_path, start_line;

-- find_imports: Find import/include statements.
-- Answers "what does this file depend on?"
--
-- Examples:
--   SELECT * FROM find_imports('**/*.go');
CREATE OR REPLACE MACRO find_imports(file_pattern) AS TABLE
    SELECT
        file_path,
        name,
        peek AS import_statement,
        start_line
    FROM read_ast(file_pattern, ignore_errors := true)
    WHERE is_import(semantic_type)
    ORDER BY file_path, start_line;

-- find_in_ast: Search AST by semantic category (generalized).
-- Replaces separate queries for calls, imports, loops, etc.
--
-- Supported kinds:
--   'calls'        — function/method call sites
--   'imports'      — import/include statements
--   'definitions'  — all named definitions
--   'loops'        — loop constructs
--   'conditionals' — if/switch/ternary
--   'strings'      — string literals
--   'comments'     — comments and docstrings
--
-- Examples:
--   SELECT * FROM find_in_ast('**/*.go', 'calls', 'AppendRaw%');
--   SELECT * FROM find_in_ast('**/*.go', 'imports');
CREATE OR REPLACE MACRO find_in_ast(file_pattern, kind, name_pattern := '%') AS TABLE
    SELECT
        file_path,
        name,
        start_line,
        peek AS context
    FROM read_ast(file_pattern, ignore_errors := true)
    WHERE name LIKE name_pattern
      AND CASE kind
          WHEN 'calls'        THEN is_call(semantic_type)
          WHEN 'imports'      THEN is_import(semantic_type)
          WHEN 'definitions'  THEN is_definition(semantic_type)
          WHEN 'loops'        THEN is_loop(semantic_type)
          WHEN 'conditionals' THEN is_conditional(semantic_type)
          WHEN 'strings'      THEN is_string_literal(semantic_type)
          WHEN 'comments'     THEN is_comment(semantic_type)
          ELSE false
          END
    ORDER BY file_path, start_line;

-- code_structure: Structural overview of files with complexity metrics.
-- Answers "which functions are large or complex?" Use BEFORE reading code.
-- Shows top-level definitions with size and cyclomatic complexity for triage.
--
-- Examples:
--   SELECT * FROM code_structure('internal/cache/*.go');
CREATE OR REPLACE MACRO code_structure(file_pattern) AS TABLE
    WITH ast AS (
        SELECT * FROM read_ast(file_pattern, ignore_errors := true)
    ),
    defs AS (
        SELECT
            file_path, name, node_id, semantic_type,
            start_line, end_line, descendant_count, children_count
        FROM ast
        WHERE is_definition(semantic_type)
          AND name != ''
          AND depth <= 2
    ),
    func_complexity AS (
        SELECT
            d.node_id,
            d.file_path,
            count(CASE WHEN is_conditional(n.semantic_type)
                AND (n.type LIKE '%_statement' OR n.type LIKE '%_clause'
                     OR n.type LIKE '%_expression' OR n.type LIKE '%_arm'
                     OR n.type LIKE '%_case' OR n.type LIKE '%_branch')
                THEN 1 END) AS conditionals,
            count(CASE WHEN is_loop(n.semantic_type)
                AND (n.type LIKE '%_statement' OR n.type LIKE '%_expression'
                     OR n.type LIKE '%_loop')
                THEN 1 END) AS loops
        FROM defs d
        JOIN ast n
          ON n.node_id > d.node_id
         AND n.node_id <= d.node_id + d.descendant_count
         AND n.file_path = d.file_path
        WHERE is_function_definition(d.semantic_type)
        GROUP BY d.node_id, d.file_path
    )
    SELECT
        d.file_path,
        d.name,
        semantic_type_to_string(d.semantic_type) AS kind,
        d.start_line,
        d.end_line,
        d.end_line - d.start_line + 1 AS line_count,
        d.descendant_count,
        d.children_count,
        CASE WHEN is_function_definition(d.semantic_type)
             THEN fc.conditionals + fc.loops + 1
             ELSE NULL END AS cyclomatic_complexity
    FROM defs d
    LEFT JOIN func_complexity fc
      ON d.node_id = fc.node_id AND d.file_path = fc.file_path
    ORDER BY d.file_path, d.start_line;

-- complexity_hotspots: Most complex functions ranked by cyclomatic complexity.
-- Answers "what is hardest to understand / most in need of review?"
--
-- Examples:
--   SELECT * FROM complexity_hotspots('**/*.go');
--   SELECT * FROM complexity_hotspots('internal/**/*.go', 10);
CREATE OR REPLACE MACRO complexity_hotspots(file_pattern, n := 20) AS TABLE
    WITH ast AS (
        SELECT * FROM read_ast(file_pattern, ignore_errors := true)
    ),
    funcs AS (
        SELECT node_id, file_path, name, start_line, end_line,
               depth AS func_depth, descendant_count
        FROM ast
        WHERE is_function_definition(semantic_type)
          AND name IS NOT NULL AND name != ''
    ),
    func_metrics AS (
        SELECT
            f.node_id, f.file_path, f.name, f.start_line, f.end_line,
            f.end_line - f.start_line + 1 AS lines,
            count(CASE WHEN n.type = 'return_statement' THEN 1 END) AS return_count,
            count(CASE WHEN is_conditional(n.semantic_type)
                AND (n.type LIKE '%_statement' OR n.type LIKE '%_clause'
                     OR n.type LIKE '%_expression' OR n.type LIKE '%_arm'
                     OR n.type LIKE '%_case' OR n.type LIKE '%_branch')
                THEN 1 END) AS conditionals,
            count(CASE WHEN is_loop(n.semantic_type)
                AND (n.type LIKE '%_statement' OR n.type LIKE '%_expression'
                     OR n.type LIKE '%_loop')
                THEN 1 END) AS loops,
            COALESCE(CAST(max(n.depth) AS INTEGER) - CAST(f.func_depth AS INTEGER), 0) AS max_depth
        FROM funcs f
        LEFT JOIN ast n
          ON n.node_id > f.node_id
         AND n.node_id <= f.node_id + f.descendant_count
         AND n.file_path = f.file_path
        GROUP BY f.node_id, f.file_path, f.name, f.start_line, f.end_line, f.func_depth
    )
    SELECT
        file_path, name, lines,
        conditionals + loops + 1 AS cyclomatic,
        conditionals, loops, return_count, max_depth
    FROM func_metrics
    ORDER BY cyclomatic DESC
    LIMIT n;

-- function_callers: All call sites for a named function across a codebase.
-- Answers "who calls X?" with the enclosing function for each call site.
--
-- Examples:
--   SELECT * FROM function_callers('**/*.go', 'AppendRaw');
CREATE OR REPLACE MACRO function_callers(file_pattern, func_name) AS TABLE
    WITH calls AS (
        SELECT file_path, start_line, node_id
        FROM read_ast(file_pattern, ignore_errors := true)
        WHERE is_call(semantic_type) AND name = func_name
    ),
    enclosing AS (
        SELECT file_path, name,
               semantic_type_to_string(semantic_type) AS kind,
               start_line AS def_start,
               end_line   AS def_end
        FROM read_ast(file_pattern, ignore_errors := true)
        WHERE is_definition(semantic_type)
          AND semantic_type_to_string(semantic_type)
              IN ('DEFINITION_FUNCTION', 'DEFINITION_CLASS', 'DEFINITION_MODULE')
          AND name != ''
    ),
    matched AS (
        SELECT
            c.file_path,
            c.start_line AS call_line,
            e.name        AS caller_name,
            e.kind        AS caller_kind,
            e.def_end - e.def_start AS scope_size,
            row_number() OVER (
                PARTITION BY c.file_path, c.start_line
                ORDER BY e.def_end - e.def_start
            ) AS rn
        FROM calls c
        LEFT JOIN enclosing e
          ON c.file_path = e.file_path
         AND c.start_line BETWEEN e.def_start AND e.def_end
    )
    SELECT file_path, call_line, caller_name, caller_kind
    FROM matched
    WHERE rn = 1
    ORDER BY file_path, call_line;

-- module_dependencies: Internal import relationships across a codebase.
-- Shows which modules import which, with fan-in count.
-- Note: uses Python-style 'from X import Y' regex — adapt peek regex for other
-- import syntaxes (Go: peek LIKE '"%package_prefix%"').
--
-- Examples:
--   SELECT * FROM find_imports('**/*.go') WHERE name LIKE 'github.com/myorg/%';
CREATE OR REPLACE MACRO module_dependencies(file_pattern, package_prefix) AS TABLE
    WITH raw_imports AS (
        SELECT DISTINCT
            file_path,
            regexp_extract(
                peek,
                'from (' || package_prefix || '[a-zA-Z0-9_.]*)',
                1
            )::VARCHAR AS target_module
        FROM read_ast(file_pattern, ignore_errors := true)
        WHERE is_import(semantic_type)
          AND peek LIKE '%from ' || package_prefix || '%'
    ),
    edges AS (
        SELECT
            replace(replace(
                regexp_extract(
                    file_path,
                    '((?:' || package_prefix || ')[a-zA-Z0-9_./]*)\.py$',
                    1
                ),
                '/', '.'
            ), '__init__', '') AS source_module,
            target_module
        FROM raw_imports
        WHERE target_module != ''
    )
    SELECT
        source_module,
        target_module,
        count(*) OVER (PARTITION BY target_module) AS fan_in
    FROM edges
    WHERE source_module != ''
    ORDER BY source_module, target_module;

-- =============================================================================
-- Git-integration macros (require duck_tails extension)
--
-- Load AFTER both extensions:
--   INSTALL duck_tails FROM community;
--   LOAD sitting_duck;
--   LOAD duck_tails;
-- =============================================================================

-- structural_diff: Compare definitions between two revisions of a file.
-- Shows functions/classes added, removed, or modified (by structural complexity,
-- not line numbers — line shifts from unrelated edits are ignored).
--
-- Requires: sitting_duck git:// URI support, duck_tails.
--
-- Examples:
--   SELECT * FROM structural_diff('internal/cache/cache.go', 'HEAD~1', 'HEAD');
CREATE OR REPLACE MACRO structural_diff(file, from_rev, to_rev, repo := '.') AS TABLE
    WITH from_defs AS (
        SELECT name, semantic_type, semantic_type_to_string(semantic_type) AS kind,
               end_line - start_line + 1 AS line_count,
               descendant_count, children_count
        FROM read_ast(git_uri(repo, file, from_rev))
        WHERE is_definition(semantic_type) AND depth <= 2 AND name != ''
    ),
    to_defs AS (
        SELECT name, semantic_type, semantic_type_to_string(semantic_type) AS kind,
               end_line - start_line + 1 AS line_count,
               descendant_count, children_count
        FROM read_ast(git_uri(repo, file, to_rev))
        WHERE is_definition(semantic_type) AND depth <= 2 AND name != ''
    )
    SELECT
        COALESCE(t.name, f.name) AS name,
        COALESCE(t.kind, f.kind) AS kind,
        CASE
            WHEN f.name IS NULL THEN 'added'
            WHEN t.name IS NULL THEN 'removed'
            WHEN t.descendant_count != f.descendant_count
              OR t.children_count   != f.children_count  THEN 'modified'
            ELSE 'unchanged'
        END AS change,
        f.line_count AS old_lines,
        t.line_count AS new_lines,
        f.descendant_count AS old_complexity,
        t.descendant_count AS new_complexity,
        COALESCE(t.descendant_count::INT, 0) - COALESCE(f.descendant_count::INT, 0) AS complexity_delta
    FROM to_defs t
    FULL OUTER JOIN from_defs f
      ON t.name = f.name AND t.semantic_type = f.semantic_type
    WHERE change != 'unchanged'
    ORDER BY change, name;

-- changed_function_summary: Functions in files changed between two revisions,
-- ranked by cyclomatic complexity. Answers "what should I review for this PR?"
--
-- Requires: duck_tails.
--
-- Examples:
--   SELECT * FROM changed_function_summary('HEAD~1', 'HEAD', '**/*.go');
CREATE OR REPLACE MACRO changed_function_summary(from_rev, to_rev, file_pattern, repo := '.') AS TABLE
    WITH changed AS (
        SELECT file_path, status
        FROM file_changes(from_rev, to_rev, repo)
        WHERE status IN ('added', 'modified')
    ),
    ast AS (
        SELECT * FROM read_ast(file_pattern, ignore_errors := true)
    ),
    metrics AS (
        SELECT * FROM ast_function_metrics(ast)
    ),
    defs AS (
        SELECT file_path, name,
               semantic_type_to_string(semantic_type) AS kind,
               start_line,
               end_line - start_line + 1 AS lines
        FROM ast
        WHERE is_definition(semantic_type) AND depth <= 2 AND name != ''
    )
    SELECT
        d.file_path, d.name, d.kind, d.lines,
        COALESCE(m.cyclomatic, 0) AS cyclomatic,
        c.status AS change_status
    FROM defs d
    JOIN changed c
      ON suffix(d.file_path, '/' || c.file_path) OR d.file_path = c.file_path
    LEFT JOIN metrics m ON d.file_path = m.file_path AND d.name = m.name
    ORDER BY COALESCE(m.cyclomatic, 0) DESC, d.file_path, d.start_line;
