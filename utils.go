package websocket

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	rnd "math/rand"
	"net/http"
	"net/url"
	"strings"
)

// keyGUID (Globally Unique Identifier).
var keyGUID = []byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

func newMaskKey() [4]byte {
	n := rnd.Uint32()

	return [4]byte{byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24)}
}

func checkHeaderContains(header http.Header, key string, value string) bool {
	return header.Get(key) == value
}

func createSecret(key string) string {
	hash := sha1.New()
	hash.Write([]byte(key))
	hash.Write(keyGUID)

	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func createClientSecret() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

func extractHostPort(addr *url.URL) string {
	hostPort := addr.Host
	if strings.ContainsRune(hostPort, ':') {
		return hostPort
	}

	switch addr.Scheme {
	case "https", "wss":
		hostPort += ":443"
	default:
		hostPort += ":80"
	}

	return hostPort
}
