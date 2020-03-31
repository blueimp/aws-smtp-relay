FROM golang:alpine as build
# Define the target arch via docker build --platform and --build-arg arguments
ARG TARGETARCH
ARG GOARCH=$TARGETARCH
ARG GOARM
# Install git to be able to install dependencies
RUN apk --no-cache add git
WORKDIR /opt
COPY . .
# Disable CGO to build a statically compiled binary.
# ldflags explanation (see `go tool link`):
#   -s  disable symbol table
#   -w  disable DWARF generation
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/aws-smtp-relay

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /bin/aws-smtp-relay /bin/
USER 65534
ENTRYPOINT ["aws-smtp-relay"]
