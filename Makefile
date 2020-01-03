.DEFAULT_GOAL := help
FUNC_NAME?=GoStatic
DIR = $(shell pwd)

.PHONY: build

all: build upload

build: ## Compile Go code and create zip file to upload to AWS Lambda
	statik -src=./files
	env GOOS=linux GOARCH=amd64 go build -o handler main.go
	zip -j handler.zip handler

upload: ## Upload code to AWS Lambda
	aws lambda update-function-code --function-name ${FUNC_NAME} \
		--zip-file fileb://handler.zip

test: ## Test function locally. hadler is the unzipped binary. STAY_OPEN for multiple requests.
	docker run --rm  \
		-e DOCKER_LAMBDA_STAY_OPEN=1 \
		-p 9001:9001 \
		-v $(DIR):/var/task:ro,delegated \
		lambci/lambda:go1.x handler

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'