#!/bin/bash
cd pkg/apis
go run ../../vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i ./... -h ../../hack/boilerplate.go.txt
cd ../..