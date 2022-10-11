module github.com/pdbogen/mapbot

go 1.19

require (
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0
	github.com/gorilla/websocket v1.5.0
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.15
	github.com/nfnt/resize v0.0.0-20180221191011-83c6a9932646
	github.com/pkg/errors v0.9.1 // indirect
	github.com/ryanuber/go-glob v1.0.0
	github.com/sirupsen/logrus v1.9.0
	github.com/slack-go/slack v0.11.3
	golang.org/x/crypto v0.0.0-20221010152910-d6f0a8c073c2
	golang.org/x/image v0.0.0-20220902085622-e7cb96979f69
	golang.org/x/net v0.0.0-20221004154528-8021a29435af
	golang.org/x/oauth2 v0.0.0-20221006150949-b44042a4b9c1
	golang.org/x/sys v0.0.0-20221010170243-090e33056c14 // indirect
	golang.org/x/text v0.3.8 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/slack-go/slack => github.com/frozenbonito/slack v0.6.4-0.20200405161309-e7a7eaea721c
