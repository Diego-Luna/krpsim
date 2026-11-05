# krpsim
This algorithm project consists of creating a program that will optimize rendering of a process graph, with some resource constraints

## Quick CLI Reference

```bash
# Build binary
make re

# Run simulation
./krpsim <path_to_config> <delay_in_seconds>

# Run verification test suite with race detector
make teste
# or
go test -v -race -count=1 ./src/...

# Run static analysis and formatting checks
go vet ./src/...
gofmt -l src
```
