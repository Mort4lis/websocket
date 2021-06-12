package handlers

import "net/http"

func RegisterHTTPHandlers(router *http.ServeMux) {
	router.HandleFunc("/", initWebsocket)
}
