package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/pdbogen/mapbot/common/blobserv"
	_ "github.com/pdbogen/mapbot/common/cache"
	"github.com/pdbogen/mapbot/common/db"
	"github.com/pdbogen/mapbot/common/db/anydb"
	mbLog "github.com/pdbogen/mapbot/common/log"
	helpController "github.com/pdbogen/mapbot/controller/help"
	"github.com/pdbogen/mapbot/controller/mapController"
	markCtrl "github.com/pdbogen/mapbot/controller/mark"
	maskController "github.com/pdbogen/mapbot/controller/mask"
	tokenController "github.com/pdbogen/mapbot/controller/token"
	"github.com/pdbogen/mapbot/controller/web"
	workflowController "github.com/pdbogen/mapbot/controller/workflow"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/context/databaseContext"
	"github.com/pdbogen/mapbot/model/types"
	httpUi "github.com/pdbogen/mapbot/ui/http"
	"github.com/pdbogen/mapbot/ui/slack"
	"golang.org/x/crypto/acme/autocert"
	"net/http"
)

var log = mbLog.Log

var backends = []string{"postgres"}

func main() {
	SlackClientToken := flag.String("slack-client-id", "", "slack Client ID")
	SlackClientSecret := flag.String("slack-client-secret", "", "slack Client Secret")
	SlackVerificationToken := flag.String("slack-verification-token", "", "slack verification token; if unset or incorrect, message actions will not work")
	Domain := flag.String("domain", "map.haversack.io", "domain name to receive redirects, construct URLs, and request ACME certs")
	Port := flag.Int("port", 8443, "port to listen on for web requests and slack oauth responses")
	Tls := flag.Bool("tls", false, "if set, mapbot will use ACME to obtain a cert from Let's Encrypt for the above-configured domain")
	DbType := flag.String("db-type", "postgres", "database type to use: postgres or sqlite3")
	ESKey := flag.String("elephant-sql-key", "", "API Key for elephant sql, to create/access DB in lieu of --db-* parameters")
	ESType := flag.String("elephant-sql-type", "turtle", "instance type to create on ElephantSQL; 'turtle' is free-tier (postgres)")
	DbHost := flag.String("db-host", "localhost", "fqdn of a postgresql server (postgres), or path to a sqlite3 file")
	DbPort := flag.Int("db-port", 5432, "port to use on db-host (postgres)")
	DbUser := flag.String("db-user", "postgres", "postgresql user to use for authentication (postgres)")
	DbPass := flag.String("db-pass", "postgres", "postgresql pass to use for authentication (postgres)")
	DbName := flag.String("db-name", "mapbot", "postgresql database name to use (postgres)")
	DbSsl := flag.Bool("db-ssl", true, "if set, require SSL to postgresql")
	DbReset := flag.Bool("db-reset", false, "USE WITH CARE: resets the schema by dropping ALL TABLES and re-executing migrations")
	DbResetFrom := flag.Int("db-reset-from", -1, "if 0 or greater, roll back to just before the given migration and re-apply later migrations")
	flag.Parse()

	var dbHandle anydb.AnyDb
	var err error
	switch *DbType {
	case "sqlite3":
		dbHandle, err = db.OpenSqlite3(*DbReset, *DbResetFrom, *DbHost)
	case "postgres":
		if *ESKey != "" {
			dbHandle, err = db.OpenElephant(*ESKey, *ESType, *DbReset, *DbResetFrom)
		} else {
			sslmode := "disable"
			if *DbSsl {
				sslmode = "verify-full"
			}
			dbHandle, err = db.OpenPsql(*DbHost, *DbUser, *DbPass, *DbName, *DbPort, *DbReset, *DbResetFrom, sslmode)
		}
	default:
		log.Fatalf("unrecognized db-type %q", *DbType)
	}

	if err != nil {
		log.Fatalf("unable to connect to database: %s", err)
	}

	proto := "http"
	if *Tls {
		proto = "https"
	}

	hub := &hub.Hub{}

	slackUi, err := slack.New(
		*SlackClientToken,
		*SlackClientSecret,
		dbHandle,
		proto,
		*Domain,
		*Port,
		*SlackVerificationToken,
		hub,
	)
	if err != nil {
		log.Fatalf("unable to start Slack module: %s", err)
	}

	prov := &context.ContextProvider{
		map[types.ContextType]context.ContextProviderFunc{
			"slack": slackUi.GetContext,
			"db":    databaseContext.GetContext(dbHandle),
		},
	}

	mapController.Register(hub)
	maskController.Register(hub)
	helpController.Register(hub)
	tokenController.Register(hub)
	workflowController.Register(hub)
	markCtrl.Register(hub)
	web.Register(hub, *Tls, *Domain)

	mgr := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(*Domain),
	}

	if *Tls {
		blobserv.Instance = &blobserv.BlobServ{UrlBase: "https://" + *Domain + "/blob/"}
	} else {
		blobserv.Instance = &blobserv.BlobServ{UrlBase: "http://" + *Domain + "/blob/"}
	}

	httpUi := httpUi.New(dbHandle, hub, prov, "/ui")

	router := http.NewServeMux()
	router.HandleFunc("/action", slackUi.Action)
	router.HandleFunc("/oauth", slackUi.OAuthPost)
	router.HandleFunc("/install", slackUi.OAuthAutoStart)
	router.HandleFunc("/blob/", blobserv.Instance.Serve)
	router.HandleFunc("/", slackUi.OAuthGet)
	router.Handle("/ui/", httpUi)

	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", *Port),
		Handler:   router,
		TLSConfig: &tls.Config{GetCertificate: mgr.GetCertificate},
	}
	if *Tls {
		httpChallengeServer := &http.Server{
			Addr:    ":80",
			Handler: mgr.HTTPHandler(http.RedirectHandler(fmt.Sprintf("https://%s", *Domain), http.StatusMovedPermanently)),
		}

		log.Infof("Listening for ACME challenges on http://%s:80", *Domain)
		go func() { log.Fatal(httpChallengeServer.ListenAndServe()) }()

		log.Infof("Listening on %s://%s:%d", proto, *Domain, *Port)
		log.Fatal(server.ListenAndServeTLS("", ""))
	} else {
		log.Infof("Listening on %s://%s:%d", proto, *Domain, *Port)
		log.Fatal(server.ListenAndServe())
	}
}
