FROM golang:1.10 as builder

ENV DEP_VERSION=0.4.1
RUN set -x \
 && curl -fSL -o dep "https://github.com/golang/dep/releases/download/v$DEP_VERSION/dep-linux-amd64" \
 && echo "31144e465e52ffbc0035248a10ddea61a09bf28b00784fd3fdd9882c8cbb2315 dep" | sha256sum -c - \
 && chmod +x dep \
 && mv dep $GOPATH/bin/

COPY . $GOPATH/src/github.com/ericyan/ping/
WORKDIR $GOPATH/src/github.com/ericyan/ping/

RUN set -x \
 && dep ensure -v -vendor-only \
 && go install ./cmd/pingd/

FROM gcr.io/distroless/base
COPY --from=builder /go/bin/pingd /
CMD ["/pingd"]
