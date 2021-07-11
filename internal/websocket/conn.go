package websocket

import (
	"bufio"
	"encoding/binary"
	"io"
	"net"
	"unicode/utf8"
)

// Close codes defined in RFC 6455
const (
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseNoStatusReceived        = 1005
	CloseAbnormalClosure         = 1006
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerErr       = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
)

var validReceivedCloseCodes = map[int]bool{
	CloseNormalClosure:           true,
	CloseGoingAway:               true,
	CloseProtocolError:           true,
	CloseUnsupportedData:         true,
	CloseNoStatusReceived:        false,
	CloseAbnormalClosure:         false,
	CloseInvalidFramePayloadData: true,
	ClosePolicyViolation:         true,
	CloseMessageTooBig:           true,
	CloseMandatoryExtension:      true,
	CloseInternalServerErr:       true,
	CloseServiceRestart:          true,
	CloseTryAgainLater:           true,
	CloseTLSHandshake:            false,
}

func IsValidReceivedCloseCode(code int) bool {
	return validReceivedCloseCodes[code] || (code >= 3000 && code <= 4999)
}

type Conn struct {
	conn       net.Conn
	rw         *bufio.ReadWriter
	fragFrames []Frame
}

func (ws *Conn) Receive() (frame Frame, err error) {
	defer func() {
		if err != nil && IsCloseError(err) {
			closeCode := err.(CloseError).code
			_ = ws.close(closeCode)
		}
	}()
	defer func() {
		if len(ws.fragFrames) != 0 {
			ws.fragFrames = nil
		}
	}()

	for {
		frame, err = ws.receive()
		if err != nil {
			return
		}

		switch frame.Opcode {
		case CloseOpcode:
			closeCode := CloseNormalClosure
			if len(frame.Payload) >= 2 {
				closeCode = int(binary.BigEndian.Uint16(frame.Payload[:2]))
			}

			err = ws.close(closeCode)
			if err != nil {
				return
			}
			return frame, CloseError{code: closeCode}
		case PingOpcode:
			frame.Opcode = PongOpcode
			err = ws.Send(frame)
			if err != nil {
				return
			}
		case ContinuationOpcode, TextOpcode, BinaryOpcode:
			ws.fragFrames = append(ws.fragFrames, frame)
			if frame.IsFragment {
				continue
			}

			var payload []byte
			for _, fr := range ws.fragFrames {
				payload = append(payload, fr.Payload...)
			}

			frame = Frame{
				Opcode:  ws.fragFrames[0].Opcode,
				Payload: payload,
			}

			if frame.IsText() && !utf8.Valid(frame.Payload) {
				return frame, errInvalidUtf8Payload
			}
			return
		}
	}
}

func (ws *Conn) receive() (Frame, error) {
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
	switch length {
	case 126:
		lenBytes, err := ws.read(2)
		if err != nil {
			return frame, err
		}

		length = uint64(binary.BigEndian.Uint16(lenBytes))
	case 127:
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

func (ws *Conn) read(size uint64) ([]byte, error) {
	buff := make([]byte, size)
	if _, err := io.ReadFull(ws.rw, buff); err != nil {
		return nil, err
	}

	return buff, nil
}

func (ws *Conn) validate(frame Frame) error {
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

func (ws *Conn) Send(frame Frame) error {
	data := make([]byte, 2)

	data[0] = frame.Opcode
	if !frame.IsFragment {
		data[0] |= 0x80
	}

	length := uint64(len(frame.Payload))
	switch {
	case length <= 125:
		data[1] = byte(length)
	case length <= 65535:
		data[1] = 126

		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		data = append(data, lenBytes...)
	default:
		data[1] = 127

		lenBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lenBytes, length)
		data = append(data, lenBytes...)
	}

	data = append(data, frame.Payload...)
	return ws.write(data)
}

func (ws *Conn) write(data []byte) error {
	if _, err := ws.rw.Write(data); err != nil {
		return err
	}
	return ws.rw.Flush()
}

func (ws *Conn) Close() error {
	return ws.close(CloseNormalClosure)
}

func (ws *Conn) close(statusCode int) error {
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
