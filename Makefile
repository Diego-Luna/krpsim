NAME   = krpsim
VERIF  = krpsim_verif
SRCS   = cmd/krpsim/main.go \
         cmd/krpsim_verif/main.go \
         internal/config/config.go \
         internal/config/parse.go \
         internal/config/process.go \
         internal/sim/events.go \
         internal/sim/state.go \
         internal/sim/inflight.go \
         internal/solver/analysis.go \
         internal/solver/greedy.go \
         internal/solver/guards.go \
         internal/solver/priority.go \
         internal/solver/tuning.go \
         internal/trace/trace.go \
         internal/verify/verify.go \
         internal/report/report.go

all: $(NAME) $(VERIF)

$(NAME): $(SRCS)
	@go build -o $(NAME) ./cmd/krpsim

$(VERIF): $(SRCS)
	@go build -o $(VERIF) ./cmd/krpsim_verif

build: all

run: $(NAME)
	./$(NAME) $(ARGS)

test:
	@go test -v ./...

bench:
	@go test -bench=. -benchmem ./...

fmt:
	@go fmt ./...

vet:
	@go vet ./...

clean:
	@rm -f *.test *.out profile.cov

fclean: clean
	@rm -f $(NAME) $(VERIF)

re: fclean all

.PHONY: all build run test bench fmt vet clean fclean re