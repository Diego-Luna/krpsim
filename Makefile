NAME = krpsim
SRC_DIR = src
SRCS = $(wildcard $(SRC_DIR)/*.go)

# * build

all: $(NAME)

$(NAME): $(SRCS)
	@go build -o $(NAME) $(SRCS)

build: $(NAME)

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
	@rm -f $(NAME)

re: fclean all

.PHONY: all build run test bench fmt vet clean fclean re