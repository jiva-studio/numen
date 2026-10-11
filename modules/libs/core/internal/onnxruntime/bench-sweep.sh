#!/usr/bin/env bash
# Runs the device benchmarks (device_bench_test.go) over providers, compute
# units and thread counts, and writes one benchstat-ready file per
# configuration plus a header of the machine facts.
#
# Usage: bench-sweep.sh OUTDIR [MAXLOAD]
#
# MAXLOAD is the highest 1-minute load average the sweep starts at; the default
# is a quarter of the logical CPUs.
#
# Environment:
#   PROVIDERS      providers to sweep; default cpu, and cpu coreml on macOS
#   COMPUTE_UNITS  compute units swept for coreml; default all four
#   THREADS        thread counts; default 1, 2, 4, physical/2, physical, logical
#   BENCH          -bench pattern; default Device
#   COUNT          -count; default 8
#   BENCHTIME      -benchtime; default the Go default
#   NUMEN_BENCH_MODELS, NUMEN_BENCH_EMBEDDING_MODEL, NUMEN_BENCH_RUNTIME are passed through.
#
# Compare two configurations with: benchstat OUTDIR/a.txt OUTDIR/b.txt

set -euo pipefail

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: $0 OUTDIR [MAXLOAD]" >&2
  exit 2
fi
out=$1
core=$(cd "$(dirname "$0")/../.." && pwd)

os=$(uname -s)
case $os in
  Darwin)
    logical=$(sysctl -n hw.logicalcpu)
    physical=$(sysctl -n hw.physicalcpu)
    performance=$(sysctl -n hw.perflevel0.physicalcpu 2>/dev/null || echo "n/a")
    cpu=$(sysctl -n machdep.cpu.brand_string)
    memory="$(( $(sysctl -n hw.memsize) / 1024 / 1024 )) MiB"
    load=$(sysctl -n vm.loadavg | awk '{print $2}')
    default_providers="cpu coreml"
    ;;
  Linux)
    logical=$(nproc --all)
    physical=$(lscpu -p=core,socket 2>/dev/null | grep -v '^#' | sort -u | wc -l | tr -d ' ')
    performance="n/a"
    cpu=$(awk -F': ' '/^model name/ {print $2; exit}' /proc/cpuinfo)
    memory="$(awk '/^MemTotal/ {print int($2 / 1024)}' /proc/meminfo) MiB"
    load=$(awk '{print $1}' /proc/loadavg)
    default_providers="cpu"
    ;;
  *)
    echo "unsupported system: $os" >&2
    exit 2
    ;;
esac
if [ -z "$physical" ] || [ "$physical" -lt 1 ]; then physical=$logical; fi

max_load=${2:-$(awk -v n="$logical" 'BEGIN {printf "%.2f", n / 4}')}
if awk -v l="$load" -v m="$max_load" 'BEGIN {exit !(l > m)}'; then
  echo "refusing to start: the 1-minute load average is $load, above the limit $max_load" >&2
  echo "wait for the machine to settle, or pass a higher limit as the second argument" >&2
  exit 1
fi

half=$(( physical / 2 ))
threads=${THREADS:-$(printf '%s\n' 1 2 4 "$half" "$physical" "$logical" |
  awk -v max="$logical" '$1 >= 1 && $1 <= max' | sort -nu | tr '\n' ' ')}
providers=${PROVIDERS:-$default_providers}
units_for_coreml=${COMPUTE_UNITS:-ALL CPUAndNeuralEngine CPUAndGPU CPUOnly}
pattern=${BENCH:-Device}
count=${COUNT:-8}

mkdir -p "$out"
{
  echo "date:               $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "system:             $os $(uname -r) $(uname -m)"
  echo "cpu:                $cpu"
  echo "logical cpus:       $logical"
  echo "physical cores:     $physical"
  echo "performance cores:  $performance"
  echo "memory:             $memory"
  echo "uptime:             $(uptime)"
  echo "load limit:         $max_load"
  echo "go:                 $(go version)"
  echo "commit:             $(git -C "$core" rev-parse --short HEAD 2>/dev/null || echo n/a)"
  echo "providers:          $providers"
  echo "threads:            $threads"
  echo "count:              $count"
  echo "bench:              $pattern"
  echo "models:             ${NUMEN_BENCH_MODELS:-<model cache>}"
  echo "embedding model:    ${NUMEN_BENCH_EMBEDDING_MODEL:-<none>}"
  echo "runtime:            ${NUMEN_BENCH_RUNTIME:-<search order>}"
} > "$out/header.txt"
cat "$out/header.txt"

bin=$out/onnxruntime.test
(cd "$core" && go test -c -o "$bin" ./internal/onnxruntime/)

failed=0
for provider in $providers; do
  if [ "$provider" = coreml ]; then units=$units_for_coreml; else units=default; fi
  for unit in $units; do
    for t in $threads; do
      name=${provider}_${unit}_t${t}
      echo "== $name"
      compute=$unit
      if [ "$unit" = default ]; then compute=; fi
      args=(-test.run '^$' -test.bench "$pattern" -test.count "$count")
      if [ -n "${BENCHTIME:-}" ]; then args+=(-test.benchtime "$BENCHTIME"); fi
      if ! NUMEN_BENCH_PROVIDER=$provider NUMEN_BENCH_THREADS=$t NUMEN_BENCH_COMPUTE=$compute \
        "$bin" "${args[@]}" > "$out/$name.txt" 2> "$out/$name.err"; then
        echo "   failed; see $out/$name.err" >&2
        failed=1
      fi
    done
  done
done
rm -f "$bin"
exit $failed
