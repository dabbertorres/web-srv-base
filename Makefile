ZIP_TOOL := 7z
ZIP_ARGS := a

SRC_FILES := $(shell find . -type f -name "*.go")
APP_FILES := $(shell find ./app -type f)
CFG_FILES := ./cfg/db-init.sql ./cfg/web.conf

all: webServer webServer-image deployables.zip

webServer: $(SRC_FILES)
	GOOS=linux go build -tags netgo

webServer-image: webServer $(APP_FILES) Dockerfile
	docker build -t dabbertorres/web-server-base:latest .
	docker push dabbertorres/web-server-base:latest

deployables.zip: docker-compose.yml $(CFG_FILES)
	$(ZIP_TOOL) $(ZIP_ARGS) $@ $?

clean:
	rm webServer

.PHONY: all webServer-image clean
