-include .make-overrides

FLY_TF_TEST_ORG ?= fly-terraform-ci
FLY_TF_TEST_APP ?= acctestapp
FLY_TF_TEST_PARALLEL ?= 8

# Run acceptance tests
.PHONY: test
test:
	FLY_TF_TEST_ORG=$(FLY_TF_TEST_ORG) FLY_TF_TEST_APP=$(FLY_TF_TEST_APP) TF_ACC=1 go test -v -cover ./internal/provider/ -count=1 -parallel=$(FLY_TF_TEST_PARALLEL)
