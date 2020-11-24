BUILD := $(abspath ./bin)
BINARIES :=         \
	openmetricstest \

# test-impl tests an OpenMetrics implementation.
#
# To test echo parser: 
# make test-impl cmd-parser-text=echo
#
# To test github.com/prometheus/client_python parser:
# make prometheus_client_python_parser test-impl cmd-parser-text="docker run --rm -i prometheus_client_python_parser:latest"
.PHONY: test-impl
test-impl: openmetricstest
	$(BUILD)/openmetricstest -cmd-parser-text="$(cmd-parser-text)"

.PHONY: prometheus_client_python_parser
prometheus_client_python_parser:
	docker build -t prometheus_client_python_parser:latest                    \
		-f ./tests/implementations/prometheus_client_python_parser/Dockerfile \
		./tests/implementations/prometheus_client_python_parser

.PHONY: setup
setup:
	mkdir -p $(BUILD)

.PHONY: binaries
binaries: $(BINARIES)

define BINARY_RULES

.PHONY: $(BINARY)
$(BINARY): setup
	@echo Building $(BINARY)
	go build -o $(BUILD)/$(BINARY) ./src/cmd/$(BINARY)/.

endef

$(foreach BINARY,$(BINARIES),$(eval $(BINARY_RULES)))

