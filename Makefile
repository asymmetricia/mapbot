.PHONY: clean

IMAGE_URL ?= 248174752766.dkr.ecr.us-west-1.amazonaws.com/mapbot
EMOJI_MAJOR ?= 3
EMOJI_MINOR ?= 1
EMOJI_POINT ?= 2
EMOJI_VERSION = ${EMOJI_MAJOR}.${EMOJI_MINOR}.${EMOJI_POINT}

restart: .push
	ssh -At mapbot.cernu.us sudo systemctl restart mapbot

dev: .push_dev
	ssh -At mapbot.cernu.us sudo systemctl restart mapbot_dev

push: .push
.push: .docker
	@ set -e; \
	IMAGE_URL=${IMAGE_URL}; \
	aws ecr get-login-password \
	| docker login \
			--username AWS \
			--password-stdin $${IMAGE_URL%%/*}; \
	docker push ${IMAGE_URL}; \
	touch .push

push_dev: .push_dev
.push_dev: .docker
	@ set -e; \
	IMAGE_URL=${IMAGE_URL}; \
	aws ecr get-login-password \
	| docker login \
			--username AWS \
			--password-stdin $${IMAGE_URL%%/*}; \
	docker tag ${IMAGE_URL} ${IMAGE_URL}:dev; \
	docker push ${IMAGE_URL}:dev; \
	touch .push_dev

.docker: mapbot Dockerfile run.sh emoji
	docker build --pull -t mapbot .
	docker tag mapbot ${IMAGE_URL}
	touch .docker


mapbot: ${shell find -name \*.go} ui/slack/context/emoji.go static/js/main.js
	go fmt github.com/pdbogen/mapbot/...
	go generate
	CGO_ENABLED=0 go build -o mapbot

static/js/main.js: ${shell find ts/}
	docker build -t ts - < Dockerfile.ts
	docker run -v $$PWD:/work -u $$(id -u) ts -p ts

release: mapbot.darwin_amd64 mapbot.linux_amd64 mapbot.windows_amd64.exe

mapbot.darwin_amd64: mapbot
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -o mapbot.darwin_amd64

mapbot.linux_amd64: mapbot
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -o mapbot.linux_amd64

mapbot.windows_amd64.exe: mapbot
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o mapbot.windows_amd64.exe

tail:
	ssh -At mapbot.cernu.us 'for i in 1 2 3 4 5; do docker logs --tail 1 mapbot >/dev/null && exit 0; sleep $$i; done; exit 1'
	ssh -At mapbot.cernu.us docker logs -f --tail 100 mapbot

tail_dev:
	ssh -At mapbot.cernu.us 'for i in 1 2 3 4 5; do docker logs --tail 1 mapbot_dev >/dev/null && exit 0; sleep $$i; done; exit 1'
	ssh -At mapbot.cernu.us docker logs -f --tail 100 mapbot_dev

clean:
	$(RM) .push .docker mapbot

ui/slack/context/emoji.go: emoji.json
	echo 'package context' > ui/slack/context/emoji.go && \
	echo 'var emojiUrl = `https://cdn.jsdelivr.net/emojione/assets/${EMOJI_MAJOR}.${EMOJI_MINOR}/png/128/%s.png`' >> ui/slack/context/emoji.go && \
	echo 'var emojiJson = `' >> ui/slack/context/emoji.go && \
	jq . < emoji.json >> ui/slack/context/emoji.go && \
	echo '`' >> ui/slack/context/emoji.go

emoji.json: emoji.${EMOJI_VERSION}.json

emoji.${EMOJI_VERSION}.json:
	curl -f -L https://github.com/emojione/emojione-assets/raw/${EMOJI_VERSION}/emoji.json > emoji.${EMOJI_VERSION}.json
	cp emoji.${EMOJI_VERSION}.json emoji.json

# This is packed into docker as a filesystem cache for emoji
emoji: .emoji_${EMOJI_VERSION}

.emoji_${EMOJI_VERSION}:
	mkdir -p emoji
	curl -L -f https://github.com/emojione/emojione-assets/archive/${EMOJI_VERSION}.tar.gz | \
	  tar -xvz --strip-components=3 --wildcards -C emoji */png/128
	touch .emoji_${EMOJI_VERSION}

test: ui/slack/context/emoji.go
	go test -v ./...
