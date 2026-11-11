# krpsim

Schedules the processes of a resource graph so a given stock (or the total time) is optimized,
and checks the resulting trace. Subject: 42 krpsim, version 4.

## Build

```sh
make        # builds krpsim and krpsim_verif
make re     # rebuild from scratch
```

## Run

```sh
./krpsim <config> <delay>          # delay in seconds, search stops at 90% of it
./krpsim_verif <config> <trace>    # trace can be krpsim's full output, or - for stdin

./krpsim resources/simple 1 > /tmp/simple.trace
./krpsim_verif resources/simple /tmp/simple.trace
```

## Layout

Follows [Organizing a Go module](https://go.dev/doc/modules/layout): programs in `cmd/`, shared code in `internal/`.

| Path | Role |
|---|---|
| `cmd/krpsim`, `cmd/krpsim_verif` | argument handling only |
| `internal/config` | config model and parser |
| `internal/sim` | discrete-event engine: stocks, running processes, clock |
| `internal/solver` | greedy scheduler with backward priorities and catalyst guards |
| `internal/trace` | the `<cycle>:<process>` line format, shared by both programs |
| `internal/verify` | trace replay used by `krpsim_verif` and the solver tests |
| `internal/report` | everything printed to stdout |
| `configs/` | our own configs: `finite.krp` terminates, `infinite.krp` runs forever |
| `resources/` | example configs from the subject |

## Checks

```sh
make test                      # go test -v ./...
go test -race -count=1 ./...
make vet
gofmt -l cmd internal
make bench
```
