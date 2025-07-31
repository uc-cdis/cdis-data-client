FROM quay.io/cdis/golang:1.17-bullseye as build-deps

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR $GOPATH/src/github.com/calypr/data-client/

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN COMMIT=$(git rev-parse HEAD); \
    VERSION=$(git describe --always --tags); \
    printf '%s\n' 'package g3cmd'\
    ''\
    'const ('\
    '    gitcommit="'"${COMMIT}"'"'\
    '    gitversion="'"${VERSION}"'"'\
    ')' > data-client/g3cmd/gitversion.go \
    && go build -o /data-client

FROM scratch
COPY --from=build-deps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-deps /data-client /data-client
CMD ["/data-client"]
