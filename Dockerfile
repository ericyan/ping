FROM golang:1.12 as builder

ENV DEP_VERSION=0.5.1
RUN set -x \
 && curl -fSL -o dep "https://github.com/golang/dep/releases/download/v$DEP_VERSION/dep-linux-amd64" \
 && echo "7479cca72da0596bb3c23094d363ea32b7336daa5473fa785a2099be28ecd0e3 dep" | sha256sum -c - \
 && chmod +x dep \
 && mv dep $GOPATH/bin/

COPY . $GOPATH/src/github.com/ericyan/pingd/
WORKDIR $GOPATH/src/github.com/ericyan/pingd/

RUN set -x \
 && dep ensure -v -vendor-only \
 && go install ./cmd/pingd/

FROM gcr.io/distroless/base
COPY --from=builder /go/bin/pingd /
CMD ["/pingd"]
