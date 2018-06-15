SRC_FILES := $(shell find . -type f -name "*.go")
APP_FILES := $(shell find ./app -type f)

all: webServer webServer-image db-image

webServer: $(SRC_FILES)
	GOOS=linux go build -tags netgo

webServer-image: webServer $(APP_FILES) Dockerfile
	docker build -t dabbertorres/web-server-base:latest .
	docker push dabbertorres/web-server-base:latest

db-image: Dockerfile.db cfg/db-init.sql
	docker build -t dabbertorres/web-server-db:latest .
	docker push dabbertorres/web-server-db:latest

clean:
	rm webServer

.PHONY: all webServer-image db-image clean
