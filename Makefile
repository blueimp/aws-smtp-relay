# --- Variables ---

# The project name:
PROJECT=aws-smtp-relay

# The absolute path for the project's binary installation:
BIN_PATH = $(GOPATH)/bin/$(PROJECT)

# Files that require a rebuild on change:
DEPS = internal/relay/relay.go internal/smtpd/smtpd.go main.go

# GO CLI set to vgo for automatic dependency resolution:
GO_CLI ?= vgo


# --- Main targets ---

all: $(PROJECT)

# Runs the unit tests for all components:
test:
	@$(GO_CLI) test ./...

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
	all
	test \
	install \
	uninstall \
	clean

# Builds the project:
$(PROJECT): $(DEPS)
	$(GO_CLI) build

# Installs the binary in $GOPATH/bin/:
$(BIN_PATH): $(DEPS)
	$(GO_CLI) install
