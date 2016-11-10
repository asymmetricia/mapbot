package slack

import (
	"database/sql"
	"errors"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/common/rand"
	"golang.org/x/oauth2"
	SlackOAuth "golang.org/x/oauth2/slack"
)

var log = mbLog.Log

func New(id string, secret string, port int, db *sql.DB, domain string) (*SlackUi, error) {
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
		Teams: []*SlackTeam{},
		oauth: oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Endpoint:     SlackOAuth.Endpoint,
			RedirectURL:  "https://localhost",
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
	}

	log.Infof("Slack UI module ready; authorize via %s", ret.oauth.AuthCodeURL(ret.csrf[0]))
	ret.runOauth(port)
	ret.runTeams()
	return ret, nil
}

type SlackUi struct {
	clientId     string
	clientSecret string
	Teams        []*SlackTeam
	oauth        oauth2.Config
	csrf         []string
	db           *sql.DB
	domain       string
}
