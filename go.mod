module github.com/pdbogen/mapbot

go 1.13

require (
	github.com/golang/freetype v0.0.0-20161208064710-d9be45aaf745
	github.com/golang/protobuf v0.0.0-20161117033126-8ee79997227b // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/lib/pq v0.0.0-20170206200638-0477eb88c5ca
	github.com/mattn/go-sqlite3 v2.0.3+incompatible
	github.com/nfnt/resize v0.0.0-20160724205520-891127d8d1b5
	github.com/ryanuber/go-glob v0.0.0-20170128012129-256dc444b735
	github.com/shurcooL/httpfs v0.0.0-20190707220628-8d4bc4ba7749 // indirect
	github.com/shurcooL/vfsgen v0.0.0-20181202132449-6a9ea43bcacd // indirect
	github.com/sirupsen/logrus v1.5.0
	github.com/slack-go/slack v0.6.3
	github.com/stretchr/testify v1.5.1 // indirect
	golang.org/x/crypto v0.0.0-20180910181607-0e37d006457b
	golang.org/x/image v0.0.0-20170210230806-df2aa51d4407
	golang.org/x/net v0.0.0-20170211013127-61557ac0112b
	golang.org/x/oauth2 v0.0.0-20170209002143-de0725b330ab
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	google.golang.org/appengine v1.0.1-0.20170206203024-2e4a801b39fc // indirect
)

replace github.com/slack-go/slack => github.com/frozenbonito/slack v0.6.4-0.20200405161309-e7a7eaea721c
