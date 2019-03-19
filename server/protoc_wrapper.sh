#!/bin/sh
protoc -I ../proto/ ../proto/chat.proto --go_out=plugins=grpc:protobuf