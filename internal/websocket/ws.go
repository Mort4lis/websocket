package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const (
	// GUID (Globally Unique Identifier)
	GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	// Status codes
	normalClose      = 1000
	protocolError    = 1002
	noStatusReceived = 1005

	maxInt8Value   = (1 << 7) - 1
	maxUint16Value = (1 << 16) - 1
)

var handshakeRespTemplate = strings.Join([]string{
	"HTTP/1.1 101 Switching Protocols",
	"Server: go/ws-custom-server",
	"Upgrade: WebSocket",
	"Connection: Upgrade",
	"Sec-WebSocket-Accept: %s",
	"", // required for extra CRLF
	"", // required for extra CRLF
}, "\r\n")

type Websocket struct {
	conn       net.Conn
	rw         *bufio.ReadWriter
	headers    http.Header
	fragFrames []Frame
}

func NewWebsocket(w http.ResponseWriter, req *http.Request) (*Websocket, error) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, errors.New("can't get control over tcp connection")
	}

	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	return &Websocket{
		conn:    conn,
		rw:      rw,
		headers: req.Header,
	}, nil
}

func (ws *Websocket) Handshake() error {
	secret := ws.createSecret(ws.headers.Get("Sec-WebSocket-Key"))
	rawResp := fmt.Sprintf(handshakeRespTemplate, secret)

	return ws.write([]byte(rawResp))
}

func (ws *Websocket) createSecret(key string) string {
	hash := sha1.New()
	hash.Write([]byte(key))
	hash.Write([]byte(GUID))
	return base64.StdEncoding.EncodeToString(hash.Sum(nil))
}

func (ws *Websocket) Receive() (Frame, error) {
	defer func() {
		if len(ws.fragFrames) != 0 {
			ws.fragFrames = nil
		}
	}()

	for {
		frame, err := ws.receive()
		if err != nil {
			if IsCloseError(err) {
				closeCode := err.(CloseError).code
				err = ws.close(closeCode)
			}

			return frame, err
		}

		switch frame.Opcode {
		case CloseOpcode:
			closeCode := noStatusReceived
			if len(frame.Payload) >= 2 {
				closeCode = int(binary.BigEndian.Uint16(frame.Payload))
			}

			err = ws.close(closeCode)
			if err != nil {
				return frame, err
			}
			return frame, CloseError{code: closeCode, text: ""}
		case PingOpcode:
			frame.Opcode = PongOpcode
			err = ws.Send(frame)
			if err != nil {
				return frame, err
			}

			continue
		case ContinuationOpcode, TextOpcode, BinaryOpcode:
			ws.fragFrames = append(ws.fragFrames, frame)
			if frame.IsFragment {
				continue
			}

			var payload []byte
			for _, fr := range ws.fragFrames {
				payload = append(payload, fr.Payload...)
			}

			return Frame{
				Opcode:  ws.fragFrames[0].Opcode,
				Payload: payload,
			}, nil
		}
	}
}

func (ws *Websocket) receive() (Frame, error) {
	frame := Frame{}

	head, err := ws.read(2)
	if err != nil {
		return frame, err
	}

	frame.IsFragment = (head[0] & 0x80) == 0x00
	frame.Reserved = head[0] & 0x70
	frame.Opcode = head[0] & 0x0F
	frame.IsMasked = (head[1] & 0x80) == 0x80

	length := uint64(head[1] & 0x7F)
	if length == maxInt8Value-1 {
		lenBytes, err := ws.read(2)
		if err != nil {
			return frame, err
		}

		length = uint64(binary.BigEndian.Uint16(lenBytes))
	} else if length == maxInt8Value {
		lenBytes, err := ws.read(8)
		if err != nil {
			return frame, err
		}

		length = binary.BigEndian.Uint64(lenBytes)
	}
	frame.Length = length

	maskKey, err := ws.read(4)
	if err != nil {
		return frame, err
	}

	payload, err := ws.read(length)
	if err != nil {
		return frame, err
	}

	for i := uint64(0); i < uint64(len(payload)); i++ {
		payload[i] ^= maskKey[i%4]
	}
	frame.Payload = payload
	return frame, ws.validate(frame)
}

func (ws *Websocket) read(size uint64) ([]byte, error) {
	buff := make([]byte, size)
	if _, err := io.ReadFull(ws.rw, buff); err != nil {
		return nil, err
	}

	return buff, nil
}

func (ws *Websocket) validate(frame Frame) error {
	err := frame.validate()
	if err != nil {
		return err
	}

	if (frame.IsText() || frame.IsBinary()) && len(ws.fragFrames) != 0 {
		return errInvalidContinuationFrame
	}
	if frame.IsContinuation() && len(ws.fragFrames) == 0 {
		return errEmptyContinueFrames
	}
	return nil
}

func (ws *Websocket) Send(frame Frame) error {
	data := make([]byte, 2)

	data[0] = frame.Opcode
	if !frame.IsFragment {
		data[0] |= 0x80
	}

	length := uint64(len(frame.Payload))
	if length <= maxInt8Value-2 {
		data[1] = byte(length)
	} else if length <= maxUint16Value {
		data[1] = maxInt8Value - 1

		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		data = append(data, lenBytes...)
	} else {
		data[1] = maxInt8Value

		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, length)
		data = append(data, lenBytes...)
	}

	data = append(data, frame.Payload...)
	return ws.write(data)
}

func (ws *Websocket) write(data []byte) error {
	if _, err := ws.rw.Write(data); err != nil {
		return err
	}
	return ws.rw.Flush()
}

func (ws *Websocket) Close() error {
	return ws.close(normalClose)
}

func (ws *Websocket) close(statusCode int) error {
	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, uint16(statusCode))
	frame := Frame{
		Opcode:  CloseOpcode,
		Payload: payload,
	}
	if err := ws.Send(frame); err != nil {
		return err
	}

	return ws.conn.Close()
}
