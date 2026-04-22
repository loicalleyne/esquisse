-- =============================================================================
-- Canonical source: duckdb-code/macros/macros_go.sql
-- macros_go.sql — Esquisse Go-specific AST analysis macros.
--
-- TABLE macros for Go type analysis, interface discovery, and struct inspection.
-- All macros query the `ast` table that must already exist in the session.
-- Build the cache first:
--   CREATE OR REPLACE TABLE ast AS
--   SELECT * FROM read_ast('**/*.go', ignore_errors := true, peek := 300);
--
-- Prerequisites:
--   LOAD sitting_duck;
--   (ast table must exist — see above)
--
-- Usage:
--   duckdb -init /dev/null code_ast.duckdb <<'SQL'
--   LOAD sitting_duck;
--   .read scripts/macros_go.sql
--   SELECT * FROM go_struct_fields();
--   SQL
--
-- Macro index:
--   go_struct_fields()              All struct fields with their Go types
--   go_func_signatures()            All functions/methods with typed params and returns
--   go_type_flow(type_name)         Where a type is defined, referenced, and used
--   go_external_types()             External package types used and their frequency
--   go_interfaces()                 All interface definitions with method counts
--   go_interface_impls(iface_name)  Methods matching an interface name (Go: structural — verify manually)
--
-- Go AST critical rules:
--   1. node_id is per-file. All JOINs on node_id MUST also match on file_path.
--   2. Named types: `type_spec` node, not `is_class_definition()`.
--      The struct_type child carries the body; the name is on the type_spec.
--   3. Exported symbols: first character is upper-case.
--      Filter: SUBSTRING(name, 1, 1) = UPPER(SUBSTRING(name, 1, 1))
--   4. Generated files: always exclude *.pb.go and *_gen.go unless asked.
--   5. Test files: exclude file_path LIKE '%_test.go' for production analysis.
--
-- Adapted from Workflow F (Type Analysis) in the duckdb-code skill.
-- =============================================================================

-- go_struct_fields: Map all struct fields with their Go types.
-- Answers "what fields does each struct have?"
-- Traversal chain: type_spec → struct_type → field_declaration_list
--                  → field_declaration → type child
--
-- Example:
--   SELECT * FROM go_struct_fields() WHERE struct_name = 'Transcoder';
CREATE OR REPLACE MACRO go_struct_fields() AS TABLE
    WITH struct_defs AS (
        SELECT ts.name AS struct_name, st.node_id AS struct_nid, ts.file_path
        FROM ast ts
        JOIN ast st
          ON st.parent_id = ts.node_id AND st.file_path = ts.file_path
         AND st.type = 'struct_type'
        WHERE ts.type = 'type_spec'
          AND ts.file_path NOT LIKE '%.pb.go'
          AND ts.file_path NOT LIKE '%_gen.go'
    ),
    field_lists AS (
        SELECT sd.struct_name, fl.node_id AS fl_nid, sd.file_path
        FROM struct_defs sd
        JOIN ast fl
          ON fl.parent_id = sd.struct_nid AND fl.file_path = sd.file_path
         AND fl.type = 'field_declaration_list'
    ),
    fields AS (
        SELECT fl.struct_name, fd.name, fd.node_id, fd.file_path, fd.peek
        FROM field_lists fl
        JOIN ast fd
          ON fd.parent_id = fl.fl_nid AND fd.file_path = fl.file_path
         AND fd.type = 'field_declaration'
    ),
    field_types AS (
        SELECT
            f.struct_name,
            f.name           AS field_name,
            f.file_path,
            t.peek           AS field_type,
            t.type           AS type_kind
        FROM fields f
        JOIN ast t
          ON t.parent_id = f.node_id AND t.file_path = f.file_path
         AND t.type IN ('qualified_type', 'pointer_type', 'type_identifier',
                        'slice_type', 'map_type', 'array_type',
                        'interface_type', 'function_type', 'channel_type')
    )
    SELECT struct_name, field_name, field_type, type_kind, file_path
    FROM field_types
    ORDER BY struct_name, field_name;

-- go_func_signatures: All functions and methods with typed params and returns.
-- Excludes test files and generated code.
-- receiver = NULL for package-level functions, struct name for methods.
--
-- Example:
--   SELECT * FROM go_func_signatures() WHERE receiver = 'Transcoder';
CREATE OR REPLACE MACRO go_func_signatures() AS TABLE
    SELECT
        name,
        CASE WHEN type = 'method_declaration'
             THEN regexp_extract(parameters[1].name, '\*?(\w+)$', 1)
             ELSE NULL
        END AS receiver,
        signature_type AS returns,
        list_filter(parameters, lambda x: x.type != 'receiver') AS params,
        file_path,
        start_line
    FROM ast
    WHERE type IN ('function_declaration', 'method_declaration')
      AND file_path NOT LIKE '%_test.go'
      AND file_path NOT LIKE '%.pb.go'
      AND file_path NOT LIKE '%_gen.go'
    ORDER BY COALESCE(receiver, ''), name;

-- go_type_flow: Where a specific type is defined, referenced, and used.
-- Answers "who uses type X and how?"
--
-- Returns rows tagged with usage_kind:
--   DEFINED          — type_spec (the definition)
--   INTERFACE_METHOD — method in an interface
--   PARAM_TYPE       — function parameter
--   STRUCT_FIELD     — field in a struct
--   TYPE_ASSERT      — type assertion (x.(T))
--   TYPE_CONVERT     — type conversion (T(x))
--   TYPE_REF         — generic reference (field/param/local)
--   FUNC_SIGNATURE   — function return type
--   METHOD_SIGNATURE — method return type
--
-- Example:
--   SELECT * FROM go_type_flow('Transcoder') ORDER BY file_path, start_line;
CREATE OR REPLACE MACRO go_type_flow(type_name) AS TABLE
    SELECT
        CASE
            WHEN type = 'type_spec'                   THEN 'DEFINED'
            WHEN type = 'method_elem'                 THEN 'INTERFACE_METHOD'
            WHEN type = 'parameter_declaration'       THEN 'PARAM_TYPE'
            WHEN type = 'field_declaration'           THEN 'STRUCT_FIELD'
            WHEN type = 'type_assertion_expression'   THEN 'TYPE_ASSERT'
            WHEN type = 'type_conversion_expression'  THEN 'TYPE_CONVERT'
            WHEN type IN ('type_identifier',
                          'qualified_type')           THEN 'TYPE_REF'
            WHEN type = 'function_declaration'        THEN 'FUNC_SIGNATURE'
            WHEN type = 'method_declaration'          THEN 'METHOD_SIGNATURE'
            ELSE type
        END AS usage_kind,
        file_path,
        start_line,
        peek
    FROM ast
    WHERE (name = type_name OR peek ILIKE '%' || type_name || '%')
      AND type NOT IN ('identifier', 'comment',
                       'interpreted_string_literal', 'field_identifier')
    ORDER BY file_path, start_line;

-- go_external_types: External package types used across the codebase.
-- Answers "what external types does this codebase depend on most?"
-- Use to understand external dependencies before refactoring.
--
-- Example:
--   SELECT * FROM go_external_types() WHERE peek ILIKE 'arrow.%' LIMIT 20;
CREATE OR REPLACE MACRO go_external_types() AS TABLE
    SELECT
        peek        AS external_type,
        count(*)    AS usage_count,
        list(DISTINCT file_path) AS files
    FROM ast
    WHERE type = 'qualified_type'
      AND file_path NOT LIKE '%.pb.go'
      AND file_path NOT LIKE '%_gen.go'
    GROUP BY peek
    ORDER BY usage_count DESC;

-- go_interfaces: All interface definitions with method count.
-- Answers "what interfaces does this package define?"
--
-- Example:
--   SELECT * FROM go_interfaces();
CREATE OR REPLACE MACRO go_interfaces() AS TABLE
    WITH iface_nodes AS (
        SELECT ts.name AS iface_name, it.node_id AS iface_nid, ts.file_path
        FROM ast ts
        JOIN ast it
          ON it.parent_id = ts.node_id AND it.file_path = ts.file_path
         AND it.type = 'interface_type'
        WHERE ts.type = 'type_spec'
          AND ts.file_path NOT LIKE '%.pb.go'
          AND ts.file_path NOT LIKE '%_gen.go'
    ),
    method_counts AS (
        SELECT i.iface_name, i.file_path, count(m.node_id) AS method_count
        FROM iface_nodes i
        LEFT JOIN ast m
          ON m.parent_id = i.iface_nid AND m.file_path = i.file_path
         AND m.type = 'method_elem'
        GROUP BY i.iface_name, i.file_path
    )
    SELECT
        iface_name,
        method_count,
        -- Exported = first letter upper-case
        SUBSTRING(iface_name, 1, 1) = UPPER(SUBSTRING(iface_name, 1, 1)) AS exported,
        file_path
    FROM method_counts
    ORDER BY iface_name;

-- go_interface_impls: Functions/methods whose names match the methods of a
-- named interface. Because Go uses structural typing, this is a heuristic —
-- verify with the compiler. Use as a starting point for "what implements X?"
--
-- Example:
--   SELECT * FROM go_interface_impls('Writer');
CREATE OR REPLACE MACRO go_interface_impls(iface_name) AS TABLE
    WITH iface_methods AS (
        SELECT m.name AS method_name, i.file_path AS iface_file
        FROM ast ts
        JOIN ast it
          ON it.parent_id = ts.node_id AND it.file_path = ts.file_path
         AND it.type = 'interface_type'
        JOIN ast m
          ON m.parent_id = it.node_id AND m.file_path = it.file_path
         AND m.type = 'method_elem'
        WHERE ts.type = 'type_spec'
          AND ts.name = iface_name
    ),
    methods AS (
        SELECT
            name,
            regexp_extract(parameters[1].name, '\*?(\w+)$', 1) AS receiver,
            file_path,
            start_line
        FROM ast
        WHERE type = 'method_declaration'
          AND file_path NOT LIKE '%_test.go'
    )
    SELECT
        m.receiver,
        m.name   AS method_name,
        m.file_path,
        m.start_line
    FROM methods m
    JOIN iface_methods im ON m.name = im.method_name
    ORDER BY m.receiver, m.name;

-- capture_planning_context: Capture AST symbol snapshots for a task into planning_context.
-- Go-specific: uses read_ast() with sitting_duck. Excludes test files.
-- Usage: INSERT INTO planning_context SELECT * FROM capture_planning_context('P2-013', 'modify', '**/*.go', 'ProcessConfig%');
-- Note: column names (start_line, end_line, semantic_type, name, file_path, peek) and
-- parameters (ignore_errors, peek) are verified against existing macros_go.sql usage in this project.
-- Re-verify against sitting_duck docs if the extension is upgraded.
CREATE OR REPLACE MACRO capture_planning_context(task_id, role, pattern, name_like) AS TABLE
    SELECT
        task_id::VARCHAR            AS task_id,
        role::VARCHAR               AS role,
        CASE
            WHEN semantic_type = 'DEFINITION_FUNCTION' THEN 'function'
            WHEN semantic_type = 'DEFINITION_CLASS'    THEN 'type'
            ELSE 'file'
        END                         AS symbol_kind,
        name                        AS symbol_name,
        file_path,
        start_line                  AS line_start,
        end_line                    AS line_end,
        peek                        AS signature,
        now()                       AS captured_at
    FROM read_ast(pattern, ignore_errors := true, peek := 'full')
    WHERE (semantic_type = 'DEFINITION_FUNCTION' OR semantic_type = 'DEFINITION_CLASS')
      AND name ILIKE name_like
      AND file_path NOT LIKE '%_test.go';
