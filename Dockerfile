FROM alpine as build

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY ./aws-smtp-relay /bin/

ENTRYPOINT ["aws-smtp-relay"]

