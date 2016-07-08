# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/intervention-engine/multifactorriskservice

WORKDIR /go/src/github.com/intervention-engine/multifactorriskservice
RUN go get
RUN go build

# Document that the service listens on port 9000.
EXPOSE 9000
