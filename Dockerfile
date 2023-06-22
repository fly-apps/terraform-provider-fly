FROM golang:1.20-bullseye

RUN mkdir -p $GOPATH/src/github.com/fly-apps/terraform-provider-fly
WORKDIR $GOPATH/src/github.com/fly-apps/terraform-provider-fly

CMD go mod tidy; go build
