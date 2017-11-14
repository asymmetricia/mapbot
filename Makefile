.PHONY: clean

SERVER ?= vdr.cernu.us
PUSH ?= 1

restart: .push
	ssh -At vdr.cernu.us docker rm -f mapbot

push: .push
.push: .docker
	@ set -e; \
	[ "${PUSH}" -eq 0 ] && exit 0 && \
	SIZE=$$(docker inspect -s mapbot | jq '.[0].Size') && \
	docker save mapbot | pv -s $$SIZE | ssh -C vdr.cernu.us docker load && \
	touch .push

.docker: mapbot Dockerfile run.sh
	docker build -t mapbot .
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
	ssh -At vdr.cernu.us docker logs -f mapbot

clean:
	$(RM) .push .docker mapbot
