# --- Variables ---

# The project name:
PROJECT=aws-smtp-relay

# The absolute path for the project's binary installation:
BIN_PATH = $(GOPATH)/bin/$(PROJECT)

# Files that require a rebuild on change:
DIRS = internal internal/auth internal/receiver internal/receiver/aws_ses internal/relay \
	internal/relay/pinpoint internal/relay/config internal/relay/ses \
	internal/relay/filter internal/relay/client
DEPS = main.go $(wildcard *.go $(foreach fd, $(DIRS), $(fd)/*.go))


# --- Main targets ---

all: $(PROJECT)

# Runs go vet and staticcheck for all components:
lint:
	go vet ./...
	staticcheck ./...

# Runs the unit tests for all components:
test:
	go test ./...

# Installs the binary in $GOPATH/bin/:
install: $(BIN_PATH)

# Deletes the binary from $GOPATH/bin/:
uninstall:
	rm -f $(BIN_PATH)

# Removes all build artifacts:
clean:
	rm -f $(PROJECT)


# --- Helper targets ---

# Defines phony targets (targets without a corresponding target file):
.PHONY: \
	all \
	lint \
	test \
	install \
	uninstall \
	clean

# Builds the project:
$(PROJECT): $(DEPS)
	go build

# Installs the binary in $GOPATH/bin/:
$(BIN_PATH): $(DEPS)
	go install
