package main

import (
	"flag"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/ui/slack"
	"github.com/pdbogen/mapbot/common/db"
	"time"
)

var log = mbLog.Log

func main() {
	SlackClientToken := flag.String("slack-client-id", "", "slack Client ID")
	SlackClientSecret := flag.String("slack-client-secret", "", "slack Client Secret")
	SlackDomain := flag.String("slack-domain", "map.haversack.io", "domain name to receive redirects, construct URLs, and request ACME certs")
	SlackOauthPort := flag.Int("slack-oauth-port", 8443, "port on which slack UI module will receive OAuth redirects")
	DbHost := flag.String("db-host", "localhost", "fqdn of a postgresql server")
	DbPort := flag.Int("db-port", 5432, "port to use on db-host")
	DbUser := flag.String("db-user", "postgres", "postgresql user to use for authentication")
	DbPass := flag.String("db-pass", "postgres", "postgresql pass to use for authentication")
	DbName := flag.String("db-name", "mapbot", "postgresql database name to use")
	flag.Parse()

	dbHandle, err := db.Open(*DbHost, *DbUser, *DbPass, *DbName, *DbPort)
	if err != nil {
		log.Fatalf("unable to connection to database: %s", err)
	}

	if _, err := slack.New(*SlackClientToken, *SlackClientSecret, *SlackOauthPort, dbHandle, *SlackDomain); err != nil {
		log.Fatalf("unable to start Slack module: %s", err)
	}

	for {
		time.Sleep(time.Hour)
	}
}
