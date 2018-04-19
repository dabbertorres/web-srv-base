FROM alpine:3.7

EXPOSE 80/tcp
EXPOSE 443/tcp

COPY ./app       /app
COPY ./webServer /

ENTRYPOINT ["./webServer"]
