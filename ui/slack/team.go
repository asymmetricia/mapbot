package slack

import (
	"github.com/pdbogen/mapbot/model/tabula"
	"github.com/nlopes/slack"
	"fmt"
	"errors"
)

func (s *SlackUi) runTeams() error {
	results, err := s.db.Query("SELECT token FROM slack_teams")
	if err != nil {
		return fmt.Errorf("running query: %s", err)
	}
	defer results.Close()
	for results.Next() {
		cols, err := results.Columns()
		if err != nil {
			return fmt.Errorf("retrieving columns: %s", err)
		}
		if len(cols) < 1 {
			return errors.New("row with insufficient columns")
		}
		if err := s.addTeam(cols[0]); err != nil {
			return fmt.Errorf("adding team: %s", err)
		}
	}
	return nil
}

func (s *SlackUi) addTeam(token string) error {
	log.Infof("Adding team with token %s", token)
	if s.Teams == nil {
		s.Teams = []*SlackTeam{}
	}
	team := &SlackTeam{
		Channels: []SlackChannel{},
		token:    token,
		client:   slack.New(token),
	}
	team.run()
	s.Teams = append(s.Teams, team)

	_, err := s.db.Exec("INSERT INTO slack_teams VALUES ($1) ON CONFLICT DO NOTHING", token)
	return err
}

func (t *SlackTeam) run() {
	
}

type SlackTeam struct {
	Channels []SlackChannel
	token    string
	client   *slack.Client
}

type SlackChannel struct {
	ActiveTabula *tabula.Tabula
}


