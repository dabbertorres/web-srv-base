# web-srv-base
basic web server setup to build from - from scratch, in Go

# [how to run](#how-to-run)
1. Setup your target machine to run Docker.
1. [Download a release](https://github.com/dabbertorres/web-srv-base/releases/latest)
1. unzip to a directory of your choice
1. setup two docker secrets (names will be namespaced in the future), db-password and redis-password
   - useful method: `openssl rand -base64 32 | docker secret create <secret name> -`
1. modify cfg/web.conf to your liking
1. run it!
   - `docker stack deploy -c docker-compose.yml <pick a name meaningful to you>`
   - depending on which images you already have on your system, it may take a little while to setup

# customize/modify
1. Have Docker and Go installed on your machine.
1. clone the repo to your $GOPATH
1. get vendored packages
   - `dep ensure`
1. hack at the code
1. build the binary (will probably add a Makefile to do the next few steps)
   - `GOOS=linux go build -tags netgo -v`
1. build (and push the image if deploying to a different system )
   - `docker build -t <your docker namespace>/<name for your image>:<tag> .`
   - `docker push <your docker namespace>/<name for your image>`
1. change the image/tag in the docker-compose.yml to your new image
1. you're done!
   - if running remotely, copy your docker-compose.yml and cfg/ folder to the remote machine
1. see [how to run](#how-to-run)
