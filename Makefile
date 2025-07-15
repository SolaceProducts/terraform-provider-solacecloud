

TF_ACC ?= 1
SOLACE_BASE_URL ?= ${SC_API_URL} 


default: 
	go build .

build_and_test: default test

test:
	TF_ACC=$(TF_ACC) go test -v -coverprofile=coverage.out ./...

# Needs to be run with a SOLACECLOUD_API_TOKEN
testacc:
	SOLACE_BASE_URL=$(SOLACE_BASE_URL) TF_ACC=$(TF_ACC) go test -v ./... -parallel 10

autorun_full_build: 
	nodemon  -e 'go' --signal SIGTERM --exec 'make' build_and_test
