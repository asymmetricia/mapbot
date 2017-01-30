package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pdbogen/mapbot/common/db"
	mbLog "github.com/pdbogen/mapbot/common/log"
	helpController "github.com/pdbogen/mapbot/controller/help"
	"github.com/pdbogen/mapbot/controller/mapController"
	maskController "github.com/pdbogen/mapbot/controller/mask"
	tokenController "github.com/pdbogen/mapbot/controller/token"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/ui/slack"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
)

var log = mbLog.Log

func main() {
	SlackClientToken := flag.String("slack-client-id", "", "slack Client ID")
	SlackClientSecret := flag.String("slack-client-secret", "", "slack Client Secret")
	Domain := flag.String("domain", "map.haversack.io", "domain name to receive redirects, construct URLs, and request ACME certs")
	Port := flag.Int("port", 8443, "port to listen on for web requests and slack aotuh responses")
	Tls := flag.Bool("tls", false, "if set, mapbot will use ACME to obtain a cert from Let's Encrypt for the above-configured domain")
	ESKey := flag.String("elephant-sql-key", "", "API Key for elephant sql, to create/access DB in lieu of --db-* parameters")
	ESType := flag.String("elephant-sql-type", "turtle", "instance type to create on ElephantSQL; 'turtle' is free-tier")
	DbHost := flag.String("db-host", "localhost", "fqdn of a postgresql server")
	DbPort := flag.Int("db-port", 5432, "port to use on db-host")
	DbUser := flag.String("db-user", "postgres", "postgresql user to use for authentication")
	DbPass := flag.String("db-pass", "postgres", "postgresql pass to use for authentication")
	DbName := flag.String("db-name", "mapbot", "postgresql database name to use")
	DbReset := flag.Bool("db-reset", false, "USE WITH CARE: resets the schema by dropping ALL TABLES and re-executing migrations")
	flag.Parse()

	var dbHandle *sql.DB
	var err error
	if *ESKey != "" {
		dbHandle, err = db.OpenElephant(*ESKey, *ESType, *DbReset)
	} else {
		dbHandle, err = db.Open(*DbHost, *DbUser, *DbPass, *DbName, *DbPort, *DbReset)
	}
	if err != nil {
		log.Fatalf("unable to connection to database: %s", err)
	}

	proto := "http"
	if *Tls {
		proto = "https"
	}

	hub := &hub.Hub{}
	mapController.Register(hub)
	maskController.Register(hub)
	helpController.Register(hub)
	tokenController.Register(hub)

	slackUi, err := slack.New(
		*SlackClientToken,
		*SlackClientSecret,
		dbHandle,
		proto,
		*Domain,
		*Port,
		hub,
	)
	if err != nil {
		log.Fatalf("unable to start Slack module: %s", err)
	}

	mgr := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*Domain),
	}
	router := mux.NewRouter()
	router.HandleFunc("/oauth", slackUi.OAuthPost)
	router.HandleFunc("/", slackUi.OAuthGet)
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", *Port),
		Handler:   router,
		TLSConfig: &tls.Config{GetCertificate: mgr.GetCertificate},
	}
	log.Infof("Listening on %s://%s:%d", proto, *Domain, *Port)
	if *Tls {
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
