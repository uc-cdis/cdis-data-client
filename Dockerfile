FROM golang:1.10 as build-deps

WORKDIR /gen3client
ENV GOPATH=/gen3client

RUN go get github.com/mitchellh/go-homedir \
    github.com/spf13/cobra \
    github.com/spf13/viper

COPY . /gen3client

# Populate git version info into the code
RUN echo "package g3cmd\n\nconst (" >src/g3cmd/gitversion.go \
    && COMMIT=`git rev-parse HEAD` && echo "    gitcommit=\"${COMMIT}\"" >>src/g3cmd/gitversion.go \
    && VERSION=`git describe --always --tags` && echo "    gitversion=\"${VERSION}\"" >>src/g3cmd/gitversion.go \
    && echo ")" >>src/g3cmd/gitversion.go

RUN go build -ldflags "-linkmode external -extldflags -static" -o gen3-client

# Store only the resulting binary in the final image
# Resulting in significantly smaller docker image size
FROM scratch
COPY --from=build-deps /gen3client/gen3-client /gen3-client
CMD ["/gen3-client"]
