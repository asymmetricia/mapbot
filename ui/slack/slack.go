package slack

import (
	"database/sql"
	"errors"
	"fmt"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/common/rand"
	"github.com/pdbogen/mapbot/hub"
	"golang.org/x/oauth2"
	SlackOAuth "golang.org/x/oauth2/slack"
	"sync"
	"reflect"
)

var log = mbLog.Log

func New(id string, secret string, db *sql.DB, proto string, domain string, port int, botHub *hub.Hub) (*SlackUi, error) {
	if id == "" {
		return nil, errors.New("client ID must not be blank")
	}
	if secret == "" {
		return nil, errors.New("client secret must not be blank")
	}
	if db == nil {
		return nil, errors.New("db handle must be non-nil")
	}
	ret := &SlackUi{
		Teams: []*Team{},
		oauth: oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Endpoint:     SlackOAuth.Endpoint,
			RedirectURL:  fmt.Sprintf("%s://%s:%d/oauth", proto, domain, port),
			Scopes: []string{
				"bot",
				"files:write:user",
				"commands",
			},
		},
		csrf: []string{
			rand.RandHex(32),
		},
		db:     db,
		domain: domain,
		botHub: botHub,
	}

	log.Info("Slack UI module ready")
	if err := ret.runTeams(); err != nil {
		return nil, err
	}

	botHub.Subscribe(hub.CommandType("howdy"), CmdHowdy)

	return ret, nil
}

func CmdHowdy(c *hub.Command) {
	if cc, ok := c.Context.(*CommandContext); ok {
		cc.Team.Respond(cc, "Howdy!")
	} else {
		log.Errorf("received howdy with unknown context type %s", reflect.TypeOf(c.Context))
	}
}

type SlackUi struct {
	clientId     string
	clientSecret string
	Teams        []*Team
	oauth        oauth2.Config
	csrf         []string
	db           *sql.DB
	domain       string
	teamWg       sync.WaitGroup
	botHub       *hub.Hub
}
