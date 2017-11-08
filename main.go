package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/common/db/anydb"
	mbLog "github.com/pdbogen/mapbot/common/log"
	helpController "github.com/pdbogen/mapbot/controller/help"
	"github.com/pdbogen/mapbot/controller/mapController"
	maskController "github.com/pdbogen/mapbot/controller/mask"
	tokenController "github.com/pdbogen/mapbot/controller/token"
	workflowController "github.com/pdbogen/mapbot/controller/workflow"
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
	DbType := flag.String("db-type", "postgres", "database type to use: postgres, sqlite3, or inmemory")
	ESKey := flag.String("elephant-sql-key", "", "API Key for elephant sql, to create/access DB in lieu of --db-* parameters")
	ESType := flag.String("elephant-sql-type", "turtle", "instance type to create on ElephantSQL; 'turtle' is free-tier (postgres)")
	DbHost := flag.String("db-host", "localhost", "fqdn of a postgresql server (postgres)")
	DbPort := flag.Int("db-port", 5432, "port to use on db-host (postgres)")
	DbUser := flag.String("db-user", "postgres", "postgresql user to use for authentication (postgres)")
	DbPass := flag.String("db-pass", "postgres", "postgresql pass to use for authentication (postgres)")
	DbName := flag.String("db-name", "mapbot", "postgresql database name to use (postgres)")
	DbReset := flag.Bool("db-reset", false, "USE WITH CARE: resets the schema by dropping ALL TABLES and re-executing migrations")
	DbResetFrom := flag.Int("db-reset-from", -1, "if 0 or greater, roll back to just before the given migration and re-apply later migrations")
	flag.Parse()

	var dbHandle anydb.AnyDb
	var err error
	switch *DbType {
	case "inmemory":
		dbHandle, err = db.OpenInMemory(*DbReset, *DbResetFrom)
	case "postgres":
		if *ESKey != "" {
			dbHandle, err = db.OpenElephant(*ESKey, *ESType, *DbReset, *DbResetFrom)
		} else {
			dbHandle, err = db.OpenPsql(*DbHost, *DbUser, *DbPass, *DbName, *DbPort, *DbReset, *DbResetFrom)
		}
	}

	if err != nil {
		log.Fatalf("unable to connect to database: %s", err)
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
	workflowController.Register(hub)

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
