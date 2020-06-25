# NB(rob): Move this to top level Makefile once merged to master, 
# then can add a rule to CircleCI to build the protobuf to make sure
# it is valid.
%.pb.go: %.proto
	protoc --go_out=. $<

all: $(patsubst %.proto,%.pb.go,$(wildcard *.proto))
