package slack

import (
	"fmt"
	"net/http"
	"golang.org/x/net/context"
)

func (ui *SlackUi) runOauth(port int) {
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(ui.receiveOauth),
	}
	log.Infof("Slack UI listening for OAuth redirects on :%d", port)
	go log.Fatal(server.ListenAndServe())
}

func (ui *SlackUi) receiveOauth(rw http.ResponseWriter, req *http.Request) {
	code := req.FormValue("code")
	state := req.FormValue("state")
	if code == "" {
		log.Error("received request with no 'code'")
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("no code provided"))
		return
	}
	if state == "" {
		log.Error("received request with no 'state'")
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("no state provided"))
		return
	}

	found := false
	for _, existingState := range ui.csrf {
		if state == existingState {
			found = true
			break
		}
	}
	if !found {
		log.Errorf("received request with invalid state %q", state)
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("invalid state"))
		return
	}

	token, err := ui.oauth.Exchange(context.Background(), code)
	if err != nil {
		log.Errorf("failed to exchange token: %s", err)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("token exchange failed"))
		return
	}

	// Note that, per https://api.slack.com/docs/oauth, slack access tokens do not currently expire
	if err := ui.addTeam(token.AccessToken); err != nil {
		log.Errorf("saving to DB: %s", err)
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte("error saving token"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("done!"))
}

