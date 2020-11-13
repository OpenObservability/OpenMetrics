%.pb.go: %.proto
	protoc --go_out=plugins=grpc:. $<

all: $(patsubst %.proto,%.pb.go,$(wildcard *.proto))
