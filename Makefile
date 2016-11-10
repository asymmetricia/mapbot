run: .run
.run: .docker

.docker: mapbot Dockerfile
	docker build -t mapbot Dockerfile

mapbot: ${shell find -name \*.go}
	go build -o mapbot .
