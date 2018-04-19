FROM alpine:3.7

EXPOSE 80/tco
EXPOSE 443/tcp

COPY ./webServer /

ENTRYPOINT ["./webServer"]
