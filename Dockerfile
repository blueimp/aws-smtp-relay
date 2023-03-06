FROM alpine as build

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY ./aws-smtp-relay /bin/

USER 65534
ENTRYPOINT ["aws-smtp-relay"]

