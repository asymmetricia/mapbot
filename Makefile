SERVER ?= vdr.cernu.us

restart: .push
	ssh -At vdr.cernu.us sudo systemctl restart mapbot

push: .push
.push: .docker
	SIZE=$$(docker inspect -s mapbot | jq '.[0].Size'); \
  docker save mapbot | pv -s $$SIZE | ssh -C vdr.cernu.us docker load
	touch .push

.docker: mapbot Dockerfile run.sh
	docker build -t mapbot .
	touch .docker


mapbot: ${shell find -name \*.go}
	go build -o mapbot

release:
	GOOS=darwin  GOARCH=amd64 go build -o mapbot.darwin_amd64
	GOOS=linux   GOARCH=amd64 go build -o mapbot.linux_amd64
	GOOS=windows GOARCH=amd64 go build -o mapbot.windows_amd64.exe
