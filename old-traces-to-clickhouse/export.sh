#!/bin/bash

# port-forward the old/source jaeger-query service
# kubectl port-forward svc/pearjet-observe-backend-jaeger-query 16686:16686

curl -XGET http://localhost:16686/api/services | jq -c '.data[]' | tr -d '"' | while read i; do
  echo Fetching traces for service $i; 
  curl -XGET http://localhost:16686/api/traces?service=$i | jq . > traces-${i}.json
done
