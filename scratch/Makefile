EXEC_NAME = xPlane
PROTO_DIR = dynspec
BUILD_DIR = build

.PHONY: all
all: build

app: ${PROTO_DIR}/common.proto ${PROTO_DIR}/policy.proto ${PROTO_DIR}/query.proto
	rm -rf app
	mkdir -p app
	protoc --proto_path=${PROTO_DIR} \
		--go_out=./app --go_opt=paths=source_relative \
    	--go-grpc_out=./app --go-grpc_opt=paths=source_relative \
		$^

.PHONY: build
build: app
	mkdir -p build
	go build -o ${BUILD_DIR}/${EXEC_NAME} ./cmd/main.go

.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}