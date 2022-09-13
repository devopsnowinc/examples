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

## Current Issue(s)

**UPDATE: RESOLVED** (see commit history)

Right now, after importing old traces to a new instance, I'm seeing jaeger-query throw `500` when looking for those old traces:

```
{"level":"error","ts":1662771900.364427,"caller":"app/http_handler.go:487","msg":"HTTP handler, Internal Server Error","error":"stream error: rpc error: code = Unknown desc = invalid length for TraceID",
```

I believe this has to do with the encoding of the traceID within the jaeger_spans_local.models -  we may have to encode as `protobuf` (I'll dig this further)

**OPEN:**

After importing old traces, the jaeger-querier seems to `panic` due to clock-skew adjustments:

```
{"level":"error","ts":1663032069.7239766,"caller":"recoveryhandler/zap.go:33","msg":"runtime error: invalid memory address or nil pointer dereference","stacktrace":"github.com/jaegertracing/jaeger/pkg/recoveryhandler.zapRecoveryWrapper.Println\n\tgithub.com/jaegertracing/jaeger/pkg/recoveryhandler/zap.go:33\ngithub.com/gorilla/handlers.recoveryHandler.log\n\tgithub.com/gorilla/handlers@v1.5.1/recovery.go:83\ngithub.com/gorilla/handlers.recoveryHandler.ServeHTTP.func1\n\tgithub.com/gorilla/handlers@v1.5.1/recovery.go:74\nruntime.gopanic\n\truntime/panic.go:838\ngithub.com/opentracing-contrib/go-stdlib/nethttp.MiddlewareFunc.func5.1\n\tgithub.com/opentracing-contrib/go-stdlib@v1.0.0/nethttp/server.go:150\nruntime.gopanic\n\truntime/panic.go:838\nruntime.panicmem\n\truntime/panic.go:220\nruntime.sigpanic\n\truntime/signal_unix.go:818\ngithub.com/jaegertracing/jaeger/model/adjuster.hostKey\n\tgithub.com/jaegertracing/jaeger/model/adjuster/clockskew.go:83\ngithub.com/jaegertracing/jaeger/model/adjuster.(*clockSkewAdjuster).buildNodesMap\n\tgithub.com/jaegertracing/jaeger/model/adjuster/clockskew.go:111\ngithub.com/jaegertracing/jaeger/model/adjuster.ClockSkew.func1\n\tgithub.com/jaegertracing/jaeger/model/adjuster/clockskew.go:43\ngithub.com/jaegertracing/jaeger/model/adjuster.Func.Adjust\n\tgithub.com/jaegertracing/jaeger/model/adjuster/adjuster.go:36\ngithub.com/jaegertracing/jaeger/model/adjuster.sequence.Adjust\n\tgithub.com/jaegertracing/jaeger/model/adjuster/adjuster.go:62\ngithub.com/jaegertracing/jaeger/cmd/query/app/querysvc.QueryService.Adjust\n\tgithub.com/jaegertracing/jaeger/cmd/query/app/querysvc/query_service.go:118\ngithub.com/jaegertracing/jaeger/cmd/query/app.(*APIHandler).convertModelToUI\n\tgithub.com/jaegertracing/jaeger/cmd/query/app/http_handler.go:353\ngithub.com/jaegertracing/jaeger/cmd/query/app.(*APIHandler).search\n\tgithub.com/jaegertracing/jaeger/cmd/query/app/http_handler.go:243\nnet/http.HandlerFunc.ServeHTTP\n\tnet/http/server.go:2084\ngithub.com/opentracing-contrib/go-stdlib/nethttp.MiddlewareFunc.func5\n\tgithub.com/opentracing-contrib/go-stdlib@v1.0.0/nethttp/server.go:154\nnet/http.HandlerFunc.ServeHTTP\n\tnet/http/server.go:2084\nnet/http.HandlerFunc.ServeHTTP\n\tnet/http/server.go:2084\ngithub.com/gorilla/mux.(*Router).ServeHTTP\n\tgithub.com/gorilla/mux@v1.8.0/mux.go:210\ngithub.com/jaegertracing/jaeger/cmd/query/app.additionalHeadersHandler.func1\n\tgithub.com/jaegertracing/jaeger/cmd/query/app/additional_headers_handler.go:28\nnet/http.HandlerFunc.ServeHTTP\n\tnet/http/server.go:2084\ngithub.com/gorilla/handlers.CompressHandlerLevel.func1\n\tgithub.com/gorilla/handlers@v1.5.1/compress.go:141\nnet/http.HandlerFunc.ServeHTTP\n\tnet/http/server.go:2084\ngithub.com/gorilla/handlers.recoveryHandler.ServeHTTP\n\tgithub.com/gorilla/handlers@v1.5.1/recovery.go:78\nnet/http.serverHandler.ServeHTTP\n\tnet/http/server.go:2916\nnet/http.(*conn).serve\n\tnet/http/server.go:1966"}
``` 

Checking to see if there are any timestamp issue in the migration.
