package main

import (
	"encoding/json"
	"github.com/gopherjs/websocket"
	"honnef.co/go/js/dom"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var uri *url.URL

func MapUrl(uri *url.URL) string {
	sessionId := uri.Query().Get("id")

	return (&url.URL{
		Scheme: uri.Scheme,
		Host:   uri.Host,
		Path:   strings.TrimRight(uri.Path, "/") + "/map",
		RawQuery: url.Values{
			"id":        []string{sessionId},
			"cachebust": []string{strconv.FormatInt(time.Now().UnixNano(), 36)},
		}.Encode(),
	}).String()
}

func WsUrl(uri *url.URL) string {
	var proto = "ws"
	if dom.GetWindow().Location().Protocol == "https" {
		proto = "wss"
	}
	sessionId := uri.Query().Get("id")

	return (&url.URL{
		Scheme: proto,
		Host:   uri.Host,
		Path:   strings.TrimRight(uri.Path, "/") + "/ws",
		RawQuery: url.Values{
			"id": []string{sessionId},
		}.Encode(),
	}).String()
}

func main() {
	var err error
	uri, err = url.Parse(dom.GetWindow().Location().String())
	if err != nil {
		panic(err)
	}

	dom.GetWindow().Document().AddEventListener("DOMContentLoaded", false, ready)
}

var ImageElement *dom.HTMLImageElement

func updateMap(payload map[string]interface{}) {
	ImageElement.Src = MapUrl(uri)
}

func connectWebsocket() {
	conn, err := websocket.Dial(WsUrl(uri))
	if err != nil {
		print("websocket to ", WsUrl(uri), " error: ", err.Error())
		return
	}
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	var message map[string]interface{}
	for {
		if err := decoder.Decode(&message); err != nil {
			print("decoding message: ", err.Error())
			return
		}
		switch cmd := message["cmd"]; cmd {
		case "update":
			updateMap(message)
		default:
			print("ignoring message: ", cmd)
		}
	}
}

func ready(dom.Event) {
	imgElem := dom.GetWindow().Document().GetElementByID("mapimage")

	if imgElem == nil {
		print("troubling, no element with id `mapimage` could be found")
		return
	}

	var ok bool
	ImageElement, ok = imgElem.(*dom.HTMLImageElement)
	if !ok {
		print("`mapimage` was found, but was ", imgElem.TagName(), ", not an image")
		return
	}

	go func() {
		for {
			connectWebsocket()
		}
	}()
}
