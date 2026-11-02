NAME   = krpsim
VERIF  = krpsim_verif
SRC_DIR = src
SRCS   = $(shell find $(SRC_DIR) -name '*.go' ! -name '*_test.go')

all: $(NAME) $(VERIF)

$(NAME): $(SRCS)
	@go build -o $(NAME) ./$(SRC_DIR)/krpsim

$(VERIF): $(SRCS)
	@go build -o $(VERIF) ./$(SRC_DIR)/krpsim_verif

build: all

run: $(NAME)
	./$(NAME) $(ARGS)

test:
	@go test -v ./$(SRC_DIR)/...

bench:
	@go test -bench=. -benchmem ./$(SRC_DIR)/...

fmt:
	@go fmt ./$(SRC_DIR)/...

vet:
	@go vet ./$(SRC_DIR)/...

clean:
	@rm -f *.test *.out profile.cov

fclean: clean
	@rm -f $(NAME) $(VERIF)

re: fclean all

.PHONY: all build run test bench fmt vet clean fclean re