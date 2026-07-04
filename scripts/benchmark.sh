#!/usr/bin/env bash
# Run DebugViz HTTP overhead benchmarks and print a summary table.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Running benchmarks (5 runs each)..."
OUT="$(mktemp)"
go test -bench='Benchmark(Handler|HTTP|Manual|Codegen)' -benchmem -count=5 ./go/lib/debugviz/ -run='^$' >"$OUT"

ns() {
  grep -E "^Benchmark$1" "$OUT" | awk '{sum+=$3; n++} END {if (n>0) printf "%.0f", sum/n; else print "0"}'
}

baseline="$(ns HandlerBaseline)"
middleware="$(ns HTTPMiddleware)"
manual="$(ns ManualSpans)"
codegen="$(ns CodegenSpans)"

pct() {
  awk -v base="$baseline" -v val="$1" 'BEGIN {
    if (base <= 0) { print "n/a"; exit }
    printf "%.1f%%", ((val - base) / base) * 100
  }'
}

rps() {
  awk -v ns="$1" 'BEGIN {
    if (ns <= 0) { print "n/a"; exit }
    printf "%.0f", 1e9 / ns
  }'
}

echo ""
echo "Mode              ns/op    RPS       overhead vs baseline"
echo "----------------  -------  --------  ---------------------"
printf "No trace          %8s %8s  —\n" "$baseline" "$(rps "$baseline")"
printf "HTTP middleware   %8s %8s  %s\n" "$middleware" "$(rps "$middleware")" "$(pct "$middleware")"
printf "Manual spans      %8s %8s  %s\n" "$manual" "$(rps "$manual")" "$(pct "$manual")"
printf "Codegen spans     %8s %8s  %s\n" "$codegen" "$(rps "$codegen")" "$(pct "$codegen")"
echo ""
echo "Raw output: $OUT"
