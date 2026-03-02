FROM quay.io/cdis/golang:1.17-bullseye AS build-deps

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR $GOPATH/src/github.com/uc-cdis/gen3-client/

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
    ')' > gen3-client/g3cmd/gitversion.go \
    && go build -o /gen3-client

RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

FROM scratch
COPY --from=build-deps /etc_passwd /etc/passwd
COPY --from=build-deps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-deps /gen3-client /gen3-client
USER nobody
CMD ["/gen3-client"]
