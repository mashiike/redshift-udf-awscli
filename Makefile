export TAG?=$(shell git log -1 --format=%h)
export AWS_ACCOUNT_ID := $(shell aws sts get-caller-identity --query 'Account' --output text)
export AWS_REGION=ap-northeast-1
export ECR=$(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com

.PHONY: build push
build:
	docker build -t $(ECR)/$(ENV)/redshift-udf-awscli:latest -t $(ECR)/$(ENV)/redshift-udf-awscli:$(TAG) -f Dockerfile .

push:
	docker push $(ECR)/$(ENV)/$*:$(TAG)
	docker push $(ECR)/$(ENV)/$*:latest
