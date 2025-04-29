#!/bin/bash

# Directory where proto files are located
PROTO_DIR="./api/proto"
# Directory for generated Go code
GO_OUT_DIR="./api"

# Create output directories if they don't exist
mkdir -p ${GO_OUT_DIR}

# Install necessary tools if not installed
if ! command -v protoc &> /dev/null
then
    echo "protoc not found, please install it"
    exit 1
fi

if ! command -v protoc-gen-go &> /dev/null
then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null
then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Generate code for auth service
protoc \
    --proto_path=${PROTO_DIR} \
    --go_out=${GO_OUT_DIR} \
    --go_opt=paths=source_relative \
    --go-grpc_out=${GO_OUT_DIR} \
    --go-grpc_opt=paths=source_relative \
    ${PROTO_DIR}/auth/v1/auth.proto

echo "Proto generation completed"