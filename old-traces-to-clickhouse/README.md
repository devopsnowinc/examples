# Migrate Existing/Old Traces to ClickHouse DB

There's a usecase that traces exist in old storage and we want to move those to new storage (in our case, ClickHouse, as well)

## The Plan

There are two parts:
-  Download/export all the traces (per service) locally
-  Run migrator tool to convert jaeger-query JSON to ClickHouse compatible model
-  Run the `.sql` on ClickHouse instance (`INSERT` statements)

## The Steps

1. Export the traces via `jaeger-query` API to get traces locally as JSON (API is per-service)

        $ kubectl port-forward -n <ns> svc/foo-bar-observe-backend-jaeger-query 16686:16686
        $ ./export.sh

2. This will create `./traces-<service>.json` files in your current directory
3. When ready, run the migrator tool (will add link to `main.go` compiled binary here) to create corresponding ClickHouse inserts

        $ ./trace-migration-tool -service console-ui --file trace-console-ui.json > import-traces.sql
        2022/09/09 18:01:18 Generating ClickHouse INSERT statements for service console-ui via file single-trace-console-ui-test-new.json...
        2022/09/09 18:01:23 Done! 

4. Copy the generated `import-traces.sql` to the clickhouse pod and run:

        $ clickhouse-client < /tmp/import.sql

## Current Issue

Right now, after importing old traces to a new instance, I'm seeing jaeger-query throw `500` when looking for those old traces:

```
{"level":"error","ts":1662771900.364427,"caller":"app/http_handler.go:487","msg":"HTTP handler, Internal Server Error","error":"stream error: rpc error: code = Unknown desc = invalid length for TraceID",
```

I believe this has to do with the encoding of the traceID within the jaeger_spans_local.models -  we may have to encode as `protobuf` (I'll dig this further)
