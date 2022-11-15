FROM golang:1.19-bullseye

WORKDIR $GOPATH/src/github.com/fly-apps/terraform-provider-fly
COPY go.mod .
RUN go mod download -x
RUN mkdir -p /out/

COPY . .
RUN go env; go install; cp $GOPATH/bin/terraform-provider-fly /out/

WORKDIR $GOPATH
