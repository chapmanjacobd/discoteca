# Performance Tracking Guide

This document describes the performance tracking and profiling tools available in discoteca.

## Overview

discoteca includes several tools for measuring and analyzing performance:

1. **Benchmark Tests** - Go standard library benchmarks for measuring operation speed
2. **pprof** - Go's built-in profiling tool for CPU, memory, and execution traces
3. **benchstat** - Statistical comparison of benchmark results
4. **Pyroscope** - Continuous profiling (optional)

## Quick Start

### Run Benchmarks

```bash
# Run all benchmarks
make benchmark

# Run benchmarks and save results
make benchmark-save

# Run specific benchmark
go test -tags "fts5" -bench=BenchmarkSearch -benchmem ./internal/commands/
```

### Generate Profiles

```bash
# Generate CPU, memory, and trace profiles
make profiles

# Generate SVG visualization of profiles
make profiles-svg
```

### Capture Screenshots

```bash
# Capture screenshots for documentation
make screenshots
```

## Benchmark Tests

### Available Benchmarks

| Benchmark | Description |
|-----------|-------------|
| `BenchmarkSearch` | Database search with various queries |
| `BenchmarkFTSSearch` | Full-text search performance |
| `BenchmarkAggregateStats` | Aggregation query performance |
| `BenchmarkHistoryQueries` | History-related queries (InProgress, Unplayed, Completed) |
| `BenchmarkAddMedia` | Media insertion performance |
| `BenchmarkMetadataExtraction` | Metadata extraction performance |

### Running Benchmarks

```bash
# Run all benchmarks for 2 seconds each
go test -tags "fts5" -bench=. -benchmem -benchtime=2s ./...

# Run specific benchmark for 10 seconds
go test -tags "fts5" -bench=BenchmarkSearch -benchmem -benchtime=10s ./internal/commands/

# Run with CPU profiling
go test -tags "fts5" -bench=BenchmarkSearch -cpuprof=cpu.prof -memprof=mem.prof ./internal/commands/
```

### Comparing Results with benchstat

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks on base branch
go test -tags "fts5" -bench=. -benchmem -count=5 ./... > old.txt

# Run benchmarks on current branch
go test -tags "fts5" -bench=. -benchmem -count=5 ./... > new.txt

# Compare results
benchstat old.txt new.txt
```

Example output:
```
name                old time/op    new time/op    delta
Search/query_test   1.23ms ± 5%    1.15ms ± 3%   -6.50%  (p=0.008 n=5+5)
```

## Profiling with pprof

### CPU Profiling

```bash
# Generate CPU profile
go test -tags "fts5" -bench=BenchmarkSearch -cpuprof=cpu.prof -benchtime=30s ./internal/commands/

# View in browser
go tool pprof -http=:8080 ./internal/commands/ cpu.prof

# Generate SVG
go tool pprof -svg -output=cpu-profile.svg ./internal/commands/ cpu.prof
```

### Memory Profiling

```bash
# Generate memory profile
go test -tags "fts5" -bench=BenchmarkSearch -memprof=mem.prof -benchtime=30s ./internal/commands/

# View in browser
go tool pprof -http=:8080 ./internal/commands/ mem.prof

# Generate SVG
go tool pprof -svg -output=mem-profile.svg ./internal/commands/ mem.prof
```

### Execution Tracing

```bash
# Generate execution trace
go test -tags "fts5" -bench=BenchmarkSearch -trace=trace.out -benchtime=30s ./internal/commands/

# View in browser
go tool trace -http=:8080 trace.out

# Convert to JSON
go tool trace -json trace.out > trace.json
```

## Pyroscope Continuous Profiling

Build discoteca with Pyroscope support:

```bash
# Build with profiling tags
go build -tags "pyroscope fts5" -o disco ./cmd/disco

# Run with profiling
./disco --cpuprof=cpu.prof --memprof=mem.prof --profile-duration=60s server my.db
```

### Profiling Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--cpuprof` | Write CPU profile to file | - |
| `--memprof` | Write memory profile to file | - |
| `--trace` | Write execution trace to file | - |
| `--profile-duration` | Duration for profiling | 30s |

## CI/CD Integration

### GitHub Actions

discoteca includes two CI workflows:

1. **ci.yml** - Runs on every push/PR, includes:
   - Benchmark execution
   - Profile generation
   - Screenshot capture

2. **performance.yml** - Runs on main branch:
   - Benchmark comparison with benchstat
   - PR comments with performance changes
   - Profile artifact uploads

### Performance Regression Detection

The `performance.yml` workflow automatically:
1. Runs benchmarks on the base branch
2. Runs benchmarks on the current branch
3. Compares results with benchstat
4. Posts results as a PR comment

## Interpreting Results

### Benchmark Output

```
BenchmarkSearch/query_test-8    1000    1234567 ns/op    12345 B/op    123 allocs/op
```

- `1000` - Number of iterations
- `1234567 ns/op` - Nanoseconds per operation
- `12345 B/op` - Bytes allocated per operation
- `123 allocs/op` - Allocations per operation

### Profile Visualization

When viewing profiles in pprof:

- **Flame Graph** - Shows call stack hotspots
- **Graph View** - Shows function call relationships
- **Top Functions** - Lists most time-consuming functions
- **Call Tree** - Shows execution paths

### Performance Metrics to Watch

| Metric | Good | Warning | Action |
|--------|------|---------|--------|
| Search latency | <10ms | 10-50ms | >50ms |
| Memory allocs/op | <100 | 100-500 | >500 |
| B/op | <10KB | 10-100KB | >100KB |

## Troubleshooting

### Benchmarks are flaky

Run with `-count=5` or higher for statistical significance:
```bash
go test -bench=. -count=10 ./...
```

### Profiles are too large

Reduce benchmark duration or filter specific functions:
```bash
go tool pprof -focus=Search cpu.prof
```

### CI benchmarks fail

Check for:
- Resource contention (other processes)
- Database size differences
- Different test data

## Best Practices

1. **Run benchmarks multiple times** - Use `-count=5` minimum
2. **Compare against baseline** - Always use benchstat
3. **Profile in production-like environment** - Similar data size/hardware
4. **Look at allocations, not just time** - Memory affects GC
5. **Use traces for concurrency issues** - `go tool trace`
6. **Track trends over time** - Store benchmark results

## Additional Resources

- [Go Profiling Guide](https://go.dev/blog/pprof)
- [benchstat documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Pyroscope Documentation](https://pyroscope.io/docs/)
