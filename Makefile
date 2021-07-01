BUILD := $(abspath ./bin)

# test-impl tests an OpenMetrics implementation.
#
# To test echo parser:
# make test-impl cmd-parser-text=echo
.PHONY: test-impl
test-impl:
# openmetricstest needs to be built in /src since it requires /src/go.mod
	cd ./src && make openmetricstest
	$(BUILD)/openmetricstest -cmd-parser-text="$(cmd-parser-text)"

.PHONY: prometheus_client_python_parser
prometheus_client_python_parser:
	docker build -t prometheus_client_python_parser:latest                    \
		-f ./tests/implementations/prometheus_client_python_parser/Dockerfile \
		./tests/implementations/prometheus_client_python_parser

# To test github.com/prometheus/client_python parser:
.PHONY: test_prometheus_client_python_parser
test_prometheus_client_python_parser:
	make prometheus_client_python_parser
	make test-impl cmd-parser-text="docker run --rm -i prometheus_client_python_parser:latest"

# To test OpenMetrics validator:
.PHONY: test_open_metrics_validator
test_open_metrics_validator:
	cd ./src && make openmetricsvalidator
	make test-impl cmd-parser-text="./bin/openmetricsvalidator"


.PHONY: proto_go
proto_go: setup
	protoc --go_out=$(BUILD) --go_opt=paths=source_relative ./proto/*.proto

.PHONY: setup
setup:
	mkdir -p $(BUILD)

.PHONY: clean
clean:
	rm -rf $(BUILD)
