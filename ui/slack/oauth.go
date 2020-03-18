package slack

import (
	"fmt"
	"github.com/pdbogen/mapbot/common/rand"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"net/http"
	"reflect"
	"time"
)

func (s *SlackUi) OAuthAutoStart(rw http.ResponseWriter, req *http.Request) {
	nonce, err := s.newNonce()
	if err != nil {
		log.Errorf("error generating nonce: %s", err)
		http.Error(rw, "error generating nonce", http.StatusInternalServerError)
		return
	}

	http.Redirect(rw, req, s.oauth.AuthCodeURL(nonce), http.StatusFound)
}

func (s *SlackUi) OAuthGet(rw http.ResponseWriter, req *http.Request) {
	nonce, err := s.newNonce()
	if err != nil {
		log.Errorf("error generating nonce: %s", err)
		http.Error(rw, "error generating nonce", http.StatusInternalServerError)
		return
	}

	rw.Header().Add("content-type", "text/html")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Welcome to MapBot.<br/>"))
	rw.Write([]byte(fmt.Sprintf("<a href='%s'>Add To Slack</a>", s.oauth.AuthCodeURL(nonce))))
}

func (s *SlackUi) newNonce() (string, error) {
	nonce := rand.RandHex(32)
	_, err := s.db.Exec("INSERT INTO slack_nonces (nonce, expiry) VALUES ($1,$2)", nonce, time.Now().Add(time.Hour))
	if err != nil {
		return "", err
	}
	return nonce, nil
}

func (s *SlackUi) validateNonce(nonce string) (bool, error) {
	_, err := s.db.Exec("DELETE FROM slack_nonces WHERE expiry < $1", time.Now())
	if err != nil {
		return false, fmt.Errorf("expiring nonces: %s", err)
	}

	results, err := s.db.Query("SELECT * FROM slack_nonces WHERE nonce=$1", nonce)
	if err != nil {
		return false, fmt.Errorf("querying nonces: %s", err)
	}
	defer results.Close()

	return results.Next(), nil
}

func (s *SlackUi) invalidateNonce(nonce string) error {
	if _, err := s.db.Exec("DELETE FROM slack_nonces WHERE nonce=$1", nonce); err != nil {
		return fmt.Errorf("invalidating nonce: %s", err)
	}

	return nil
}

func (s *SlackUi) OAuthPost(rw http.ResponseWriter, req *http.Request) {
	code := req.FormValue("code")
	nonce := req.FormValue("state")
	if code == "" {
		log.Error("received request with no 'code'")
		http.Error(rw, "no code provided", http.StatusBadRequest)
		return
	}
	if nonce == "" {
		log.Error("received request with no 'state'")
		http.Error(rw, "no state provided", http.StatusBadRequest)
		return
	}

	if ok, err := s.validateNonce(nonce); err != nil {
		log.Errorf("%s: error validating nonce %q: %s", req.RemoteAddr, nonce, err)
		http.Error(rw, "error checking nonce", http.StatusInternalServerError)
		return
	} else if !ok {
		log.Errorf("%s: received request with invalid nonce %q", req.RemoteAddr, nonce)
		http.Error(rw, "invalid nonce", http.StatusBadRequest)
		return
	}

	token, err := s.oauth.Exchange(context.Background(), code)
	if err != nil {
		log.Errorf("%s: failed to exchange token: %s", req.RemoteAddr, err)
		http.Error(rw, "token exchange failed", http.StatusInternalServerError)
		return
	}

	bot_token := &BotToken{}
	if err := bot_token.FromOauthToken(token); err != nil {
		log.Errorf("oauth token did not contain bot authentication data: %s", err)
	}

	if err := s.invalidateNonce(nonce); err != nil {
		log.Errorf("%s: handling oauth redirect: %s", req.RemoteAddr, err)
		http.Error(rw, "nonce invalidation failed", http.StatusInternalServerError)
		return
	}

	s.addTeam(token.AccessToken, bot_token)

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("done!"))
}

type BotToken struct {
	BotId    string `json:"bot_user_id"`
	BotToken string `json:"bot_access_token"`
}

func (b *BotToken) FromOauthToken(t *oauth2.Token) error {
	ex := t.Extra("bot")
	if bt, ok := ex.(map[string]interface{}); ok {
		for n, d := range map[string]*string{
			"bot_access_token": &b.BotToken,
			"bot_user_id":      &b.BotId,
		} {
			if map_value, ok := bt[n]; ok {
				if string_value, ok := map_value.(string); ok {
					*d = string_value
				} else {
					return fmt.Errorf(
						"'bot' extra was map and contained %s, but its type was %s, not string",
						n,
						reflect.TypeOf(map_value))
				}
			} else {
				return fmt.Errorf("'bot' extra was map but did not contain %s", n)
			}
		}
	} else {
		return fmt.Errorf("'bot' extra was unexpected type %s", reflect.TypeOf(ex))
	}
	return nil
}

func (b *BotToken) String() string {
	if b == nil {
		return "BotToken{nil}"
	}
	return fmt.Sprintf("BotToken{BotUserId: %q, BotAccessToken: %q}", b.BotId, b.BotToken)
}
