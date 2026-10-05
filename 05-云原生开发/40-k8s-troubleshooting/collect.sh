#!/usr/bin/env bash
set -Eeuo pipefail

namespace="${1:-staging}"
selector="${2:-app=order}"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output_dir="${3:-./k8s-diagnostics-${namespace}-${timestamp}}"

umask 077
mkdir -p "$output_dir"

kubectl get deploy,rs,pods,svc -n "$namespace" -l "$selector" -o wide \
  >"$output_dir/resources.txt" 2>&1 || true
kubectl get events -n "$namespace" --sort-by=.metadata.creationTimestamp \
  >"$output_dir/events.txt" 2>&1 || true
kubectl get endpointslice -n "$namespace" \
  -l kubernetes.io/service-name=order-svc -o wide \
  >"$output_dir/endpointslices.txt" 2>&1 || true

pods="$(kubectl get pods -n "$namespace" -l "$selector" \
  -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)"

while IFS= read -r pod; do
  [[ -z "$pod" ]] && continue
  kubectl describe pod "$pod" -n "$namespace" \
    >"$output_dir/${pod}-describe.txt" 2>&1 || true
  kubectl logs "$pod" -n "$namespace" --all-containers=true \
    --timestamps=true --tail=500 \
    >"$output_dir/${pod}-logs-current.txt" 2>&1 || true
  kubectl logs "$pod" -n "$namespace" --all-containers=true --previous=true \
    --timestamps=true --tail=500 \
    >"$output_dir/${pod}-logs-previous.txt" 2>&1 || true
done <<<"$pods"

printf '诊断信息已保存到 %s\n' "$output_dir"
printf '日志可能包含用户数据，请限制访问并按保留策略清理。\n'
