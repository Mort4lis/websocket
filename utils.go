package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"net/http"
)

// keyGUID (Globally Unique Identifier).
var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

func checkHeaderContains(header http.Header, key string, value string) bool {
	return header.Get(key) == value
}

func createSecret(key string) string {
	hash := sha1.New()
	hash.Write([]byte(key))
	hash.Write(keyGUID)

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}
