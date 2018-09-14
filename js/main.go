package main

import (
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
	sessionId := uri.Query().Get("id")

	return (&url.URL{
		Scheme: "wss",
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

func ready(dom.Event) {
	go func() {
		for {
			imgElem := dom.GetWindow().Document().GetElementByID("mapimage")

			if imgElem == nil {
				print("troubling, no `mapimage` could be found")
				return
			}

			img, ok := imgElem.(*dom.HTMLImageElement)
			if !ok {
				print("`mapimage` was found, but was ", img.TagName(), ", not an image")
				return
			}

			img.Src = MapUrl(uri)

			conn, err := websocket.Dial(WsUrl(uri))
			if err == nil {
				buf := make([]byte, 1024)
				for {
					_, err := conn.Read(buf)
					if err != nil {
						break
					}
					img.Src = MapUrl(uri)
				}
			}

			print("websocket to ", WsUrl(uri), " error: ", err.Error())

			time.Sleep(time.Second)
		}
	}()
}
