package ws

import "net/http"

func initWebsocket(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("hello, world"))
}
