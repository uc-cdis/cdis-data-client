FROM golang:1.10-alpine as build-deps

RUN apk update && apk add --no-cache git ca-certificates gcc musl-dev

WORKDIR /go/src/github.com/uc-cdis/gen3-client

RUN go get github.com/mitchellh/go-homedir \
    github.com/spf13/cobra \
    github.com/spf13/viper \
    github.com/cavaliercoder/grab \
    github.com/golang/mock/gomock

COPY . .

# Populate git version info into the code
RUN printf "package g3cmd\n\nconst (" >gen3-client/g3cmd/gitversion.go \
    && COMMIT=`git rev-parse HEAD` && echo "    gitcommit=\"${COMMIT}\"" >>gen3-client/g3cmd/gitversion.go \
    && VERSION=`git describe --always --tags` && echo "    gitversion=\"${VERSION}\"" >>gen3-client/g3cmd/gitversion.go \
    && echo ")" >>gen3-client/g3cmd/gitversion.go

#RUN go test -v tests/functions_test.go

RUN go build -ldflags "-linkmode external -extldflags -static" -o bin/gen3-client

# Store only the resulting binary in the final image
# Resulting in significantly smaller docker image size
FROM scratch
COPY --from=build-deps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-deps /go/src/github.com/uc-cdis/gen3-client/bin/gen3-client /gen3-client
ENTRYPOINT ["/gen3-client"]
