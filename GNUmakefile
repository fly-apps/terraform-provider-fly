default: test

# Run acceptance tests
.PHONY: test
test:
	FLY_TF_TEST_APP=acctestapp TF_ACC=1 go test -v -cover ./internal/provider/ -count=1 -parallel=8

