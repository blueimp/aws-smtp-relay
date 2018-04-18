FROM golang:alpine as build
RUN apk --no-cache add git
RUN go get golang.org/x/vgo
WORKDIR /go/src/github.com/blueimp/aws-smtp-relay
COPY . .
# Install aws-smtp-relay as statically compiled binary:
# ldflags explanation (see `go tool link`):
#   -s  disable symbol table
#   -w  disable DWARF generation
RUN CGO_ENABLED=0 vgo install -ldflags='-s -w'

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/bin/aws-smtp-relay /bin/
USER 65534
ENTRYPOINT ["aws-smtp-relay"]
