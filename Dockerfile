FROM alpine:3.7 as certs

RUN apk update && apk add ca-certificates

FROM scratch

EXPOSE 80/tcp
EXPOSE 443/tcp

COPY --from=certs /etc/ssl/certs /etc/ssl/certs
COPY ./app /app
COPY ./webServer /

ENTRYPOINT ["./webServer"]
