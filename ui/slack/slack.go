package slack

import (
	"errors"
	"fmt"
	"github.com/pdbogen/mapbot/common/db/anydb"
	mbLog "github.com/pdbogen/mapbot/common/log"
	"github.com/pdbogen/mapbot/common/rand"
	"github.com/pdbogen/mapbot/hub"
	"github.com/pdbogen/mapbot/model/context"
	"github.com/pdbogen/mapbot/model/types"
	"golang.org/x/oauth2"
	SlackOAuth "golang.org/x/oauth2/slack"
	"strings"
	"sync"
)

var log = mbLog.Log

func New(id string, secret string, db anydb.AnyDb, proto string, domain string, port int, verificationToken string, botHub *hub.Hub) (*SlackUi, error) {
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
		oauth: oauth2.Config{
			ClientID:     id,
			ClientSecret: secret,
			Endpoint:     SlackOAuth.Endpoint,
			RedirectURL:  fmt.Sprintf("%s://%s:%d/oauth", proto, domain, port),
			Scopes: []string{
				"bot",
				"files:write:user",
				"commands",
				"team:read",
				"emoji:read",
			},
		},
		csrf: []string{
			rand.RandHex(32),
		},
		db:                db,
		domain:            domain,
		botHub:            botHub,
		verificationToken: verificationToken,
	}

	log.Info("Slack UI module ready")
	if err := ret.runTeams(); err != nil {
		return nil, err
	}

	botHub.Subscribe(hub.CommandType("user:howdy"), CmdHowdy)

	return ret, nil
}

func CmdHowdy(h *hub.Hub, c *hub.Command) {
	h.Publish(&hub.Command{
		Type:    hub.CommandType(c.From),
		Payload: "Howdy!"})
}

type SlackUi struct {
	clientId          string
	clientSecret      string
	TeamsMu           sync.RWMutex
	Teams             []*Team
	oauth             oauth2.Config
	csrf              []string
	db                anydb.AnyDb
	domain            string
	teamWg            sync.WaitGroup
	botHub            *hub.Hub
	verificationToken string
	pendingTeams      []struct {
		token string
		bot   *BotToken
	}
}

func (s *SlackUi) GetContext(id types.ContextId) (context.Context, error) {
	teamComps := strings.Split(string(id), "-")
	if len(teamComps) != 2 {
		return nil, fmt.Errorf("slack context ID expected to be TEAM-CHANNEL, but was %s", id)
	}

	s.TeamsMu.RLock()
	defer s.TeamsMu.RUnlock()
	for _, team := range s.Teams {
		if team.Info.ID == teamComps[0] {
			return team.Context(teamComps[1]), nil
		}
	}
	return nil, fmt.Errorf("team %q not found", teamComps[0])
}
