FROM alpine:3.7 as certs-source
FROM scratch

EXPOSE 80/tcp
EXPOSE 443/tcp

COPY --from=certs-source /etc/ssl/certs /etc/ssl/certs
COPY ./app /app
COPY ./webServer /

ENTRYPOINT ["./webServer"]
