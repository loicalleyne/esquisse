# Interview Guide: write-spec

## Feature Type Heuristics

Match the user's one-sentence description against signals below.
**Priority order:** api > data-model > cli > background-job > pipeline > generic.
Use the first type that matches.

| Type | Match if description contains |
|------|-------------------------------|
| `api` | endpoint, route, handler, REST, gRPC, HTTP, request, response |
| `data-model` | schema, struct, table, field, migration, model, type definition |
| `cli` | command, flag, argument, subcommand, CLI, binary, terminal |
| `background-job` | worker, cron, scheduled, retry, queue consumer, job, daemon |
| `pipeline` | stream, ingestion, transform, fan-out, Kafka, pipeline, stage, ETL |
| `generic` | none of the above |

If still ambiguous after keyword matching, ask: "Is this closer to a new API
endpoint, a new data type, a CLI command, a background worker, a data pipeline,
or something else entirely?"

---

## Question Banks

Ask all sections **in a single message**. Do not split across turns.

---

### `api` — REST/gRPC endpoint

```
## Functional
1. What is the HTTP method and path (REST) or RPC name (gRPC)?
2. What are the required request fields? What are optional?
3. What does a successful response look like? (status code + body shape)
4. Who calls this endpoint? (other services / UI / CLI / external)

## Non-Functional
5. Expected request volume (requests/sec at P50 and P99)?
6. Required latency SLO?
7. Idempotent? (same request twice = same result, no side effects)

## Error Handling
8. What are the 3 most likely error cases? What response does the caller receive?
9. Authentication required? What scheme?

## Integration
10. Which existing packages/handlers will this modify?
11. Any schema migrations required (DB, protobuf, config)?
```

---

### `data-model` — schema/struct/migration

```
## Functional
1. What is the new type or table name?
2. List all fields: name, type, required/optional, constraints (unique, non-null, etc.)
3. What existing types reference this? What references it?
4. Is this append-only, mutable, or ephemeral?

## Non-Functional
5. Expected data volume (rows/records at 1 month and 1 year)?
6. Query patterns: what indexes are needed?
7. Any PII or sensitive fields requiring special handling?

## Error Handling
8. What happens on validation failure? Where is validation enforced?
9. What is the migration strategy for existing data?

## Integration
10. Which packages/layers read this type? Which write it?
11. Any serialisation format requirements (JSON, protobuf, YAML)?
```

---

### `cli` — command/subcommand

```
## Functional
1. Full command signature: `{binary} {subcommand} [flags] [args]`
2. Required flags vs. optional flags (with defaults)?
3. What does the command do on success? What output does the user see?
4. Interactive (prompts for input) or non-interactive?

## Non-Functional
5. Expected runtime (ms / s / minutes)?
6. Should it be scriptable (machine-readable output mode, e.g. --json)?
7. Global flags it must respect (e.g. --config, --verbose)?

## Error Handling
8. What are the top 3 user error cases? What does the error message say?
9. Exit codes for each error class?

## Integration
10. Which internal packages does this command call?
11. Any config file format changes required?
```

---

### `background-job` — worker/cron/scheduler

```
## Functional
1. What triggers this job? (cron schedule / event / queue message / signal)
2. What is one unit of work? (one message / one batch / one file)
3. What does a successful run produce? (side effects, outputs, state change)
4. What is the expected runtime per unit of work?

## Non-Functional
5. Required throughput (units/sec or units/min)?
6. Concurrency: how many workers can run simultaneously?
7. Retry policy on failure (max attempts, backoff)?

## Error Handling
8. What happens when one unit fails? (skip / retry / dead-letter / halt)
9. Observability: what metrics or logs are required?

## Integration
10. What upstream system provides input? What downstream system receives output?
11. Any locking or deduplication required to prevent double-processing?
```

---

### `pipeline` — stream/ingestion/transform

```
## Functional
1. What is the source? (Kafka topic / file / HTTP stream / channel)
2. What is the sink? (DuckDB / S3 / Parquet / another topic / channel)
3. Describe one message/record in → one record out. What is the transform?
4. Batching: does the pipeline work per-message, per-batch, or per-window?

## Non-Functional
5. Required throughput (messages/sec or MB/s)?
6. Acceptable end-to-end latency (source to sink)?
7. Back-pressure strategy when sink is slow?

## Error Handling
8. What happens to a malformed message? (drop / dead-letter / halt)
9. What happens on partial batch failure?

## Integration
10. Which existing pipeline stages does this connect to?
11. Schema format of messages (protobuf / JSON / Arrow / raw bytes)?
```

---

### `generic` — fallback

```
## Functional
1. In one paragraph: what does this feature do when it works correctly?
2. What inputs does it take? What outputs does it produce?
3. What existing behaviour does it change or replace?
4. Who or what calls it, and when?

## Non-Functional
5. Any performance requirements?
6. Any concurrency requirements?
7. Any data volume requirements?

## Error Handling
8. What are the top 3 ways this can fail?
9. What does each failure mode produce (error, fallback value, halt)?

## Integration
10. Which existing packages or files will this touch?
11. Any external dependencies (new libraries, new external services)?
```
