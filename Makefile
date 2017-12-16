.PHONY: clean

SERVER ?= vdr.cernu.us
IMAGE_URL ?= 248174752766.dkr.ecr.us-west-1.amazonaws.com/mapbot

restart: .pull
	ssh -At vdr.cernu.us docker rm -f mapbot || true

push: .push
.push: .docker
	@ set -e; \
	eval "$$(aws ecr get-login)" && \
	docker push ${IMAGE_URL} && \
	touch .push

.pull: .push
	ssh ${SERVER} $$(aws ecr get-login) && \
	ssh ${SERVER} docker pull ${IMAGE_URL} && \
	ssh ${SERVER} docker tag ${IMAGE_URL} mapbot && \
	touch .pull

.docker: mapbot Dockerfile run.sh
	docker build -t mapbot .
	docker tag mapbot ${IMAGE_URL}
	touch .docker


mapbot: ${shell find -name \*.go}
	go fmt github.com/pdbogen/mapbot/...
	go build -o mapbot

release:
	go fmt github.com/pdbogen/mapbot/...
	GOOS=darwin  GOARCH=amd64 go build -o mapbot.darwin_amd64
	GOOS=linux   GOARCH=amd64 go build -o mapbot.linux_amd64
	GOOS=windows GOARCH=amd64 go build -o mapbot.windows_amd64.exe

tail:
	for i in 1 2 3 4 5; do ssh -At vdr.cernu.us docker logs --tail 1 mapbot >/dev/null && exit 0; sleep $$i; done
	ssh -At vdr.cernu.us docker logs -f --tail 100 mapbot

clean:
	$(RM) .push .docker mapbot
