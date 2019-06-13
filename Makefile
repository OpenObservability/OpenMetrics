BUILD := $(abspath ./bin)
BINARIES :=         \
	openmetricstest \

# To test "echo": 
# make test-impl cmd-parser-text=echo
# To test failure of "false":
# make test-impl cmd-parser-text=false fail=true
.PHONY: test-impl
test-impl: openmetricstest
	@if $(BUILD)/openmetricstest -cmd-parser-text $(cmd-parser-text); then \
		if [ "$(fail)" != "true" ]; then \
			echo "Passed as expected"; \
			exit 0; \
		else \
			echo "Expected failure, but succeeded"; \
			exit 1; \
		fi; \
	else \
		if [ "$(fail)" != "true" ]; then \
			echo "Failed unexpectedly"; \
			exit 1; \
		else \
			echo "Failed as expected"; \
			exit 0; \
		fi; \
	fi; \

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

