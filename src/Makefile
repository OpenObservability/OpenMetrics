BUILD := $(abspath ../bin)
BINARIES :=                \
	openmetricstest          \
	prometheusclientgoparser \
	scrapevalidator          \

.PHONY: setup
setup:
	mkdir -p $(BUILD)

.PHONY: binaries
binaries: $(BINARIES)

define BINARY_RULES

.PHONY: $(BINARY)
$(BINARY): setup
	@echo Building $(BINARY)
	go build -o $(BUILD)/$(BINARY) ./cmd/$(BINARY)/.

endef

$(foreach BINARY,$(BINARIES),$(eval $(BINARY_RULES)))
