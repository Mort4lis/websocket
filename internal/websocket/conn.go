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
	closeErr   *CloseError
	fragFrames []Frame
}

func (c *Conn) Receive() (Frame, error) {
	defer func() {
		if len(c.fragFrames) != 0 {
			c.fragFrames = nil
		}
	}()

	for {
		fr, err := c.receive()
		if err != nil {
			return fr, err
		}

		switch fr.Opcode {
		case CloseOpcode:
			closeCode := CloseNormalClosure
			if len(fr.Payload) >= 2 {
				closeCode = int(binary.BigEndian.Uint16(fr.Payload[:2]))
			}

			err = c.close(closeCode)
			if err != nil {
				return fr, err
			}

			c.closeErr = &CloseError{code: closeCode}
			return fr, c.closeErr
		case PingOpcode:
			fr.Opcode = PongOpcode
			err = c.Send(fr)
			if err != nil {
				return fr, err
			}
		case ContinuationOpcode, TextOpcode, BinaryOpcode:
			c.fragFrames = append(c.fragFrames, fr)
			if fr.IsFragment {
				continue
			}

			var payload []byte
			for _, fr = range c.fragFrames {
				payload = append(payload, fr.Payload...)
			}

			fr = Frame{
				Opcode:  c.fragFrames[0].Opcode,
				Payload: payload,
			}

			if fr.IsText() && !utf8.Valid(fr.Payload) {
				c.closeErr = errInvalidUtf8Payload
				return fr, c.closeErr
			}
			return fr, nil
		}
	}
}

func (c *Conn) receive() (Frame, error) {
	fr := Frame{}

	head, err := c.read(2)
	if err != nil {
		return fr, err
	}

	fr.IsFragment = (head[0] & 0x80) == 0x00
	fr.Reserved = head[0] & 0x70
	fr.Opcode = head[0] & 0x0F
	fr.IsMasked = (head[1] & 0x80) == 0x80

	length := uint64(head[1] & 0x7F)
	switch length {
	case 126:
		lenBytes, err := c.read(2)
		if err != nil {
			return fr, err
		}

		length = uint64(binary.BigEndian.Uint16(lenBytes))
	case 127:
		lenBytes, err := c.read(8)
		if err != nil {
			return fr, err
		}

		length = binary.BigEndian.Uint64(lenBytes)
	}
	fr.Length = length

	maskKey, err := c.read(4)
	if err != nil {
		return fr, err
	}

	payload, err := c.read(length)
	if err != nil {
		return fr, err
	}

	for i := uint64(0); i < uint64(len(payload)); i++ {
		payload[i] ^= maskKey[i%4]
	}
	fr.Payload = payload

	closeErr := c.validate(fr)
	if closeErr != nil {
		c.closeErr = closeErr
		return fr, closeErr
	}

	return fr, nil
}

func (c *Conn) read(size uint64) ([]byte, error) {
	buff := make([]byte, size)
	if _, err := io.ReadFull(c.rw, buff); err != nil {
		return nil, err
	}

	return buff, nil
}

func (c *Conn) validate(fr Frame) *CloseError {
	if fr.IsControl() && (fr.Length > 125 || fr.IsFragment) {
		return errInvalidControlFrame
	}
	if fr.Reserved > 0 {
		return errNonZeroRSVFrame
	}
	if fr.Opcode > BinaryOpcode && fr.Opcode < CloseOpcode || fr.Opcode > PongOpcode {
		return errReservedOpcodeFrame
	}
	if fr.IsClose() {
		if len(fr.Payload) >= 2 {
			code := int(binary.BigEndian.Uint16(fr.Payload[:2]))
			reason := fr.Payload[2:]
			if !IsValidReceivedCloseCode(code) {
				return errInvalidClosureCode
			}
			if !utf8.Valid(reason) {
				return errInvalidUtf8Payload
			}
		} else if len(fr.Payload) != 0 {
			return errInvalidClosurePayload
		}
	}
	if (fr.IsText() || fr.IsBinary()) && len(c.fragFrames) != 0 {
		return errInvalidContinuationFrame
	}
	if fr.IsContinuation() && len(c.fragFrames) == 0 {
		return errEmptyContinueFrames
	}
	return nil
}

func (c *Conn) Send(fr Frame) error {
	data := make([]byte, 2)

	data[0] = fr.Opcode
	if !fr.IsFragment {
		data[0] |= 0x80
	}

	length := uint64(len(fr.Payload))
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

	data = append(data, fr.Payload...)
	return c.write(data)
}

func (c *Conn) write(data []byte) error {
	if _, err := c.rw.Write(data); err != nil {
		return err
	}
	return c.rw.Flush()
}

func (c *Conn) Close() error {
	if c.closeErr != nil {
		return c.close(c.closeErr.code)
	}

	return c.close(CloseNormalClosure)
}

func (c *Conn) close(statusCode int) error {
	payload := make([]byte, 2)
	binary.BigEndian.PutUint16(payload, uint16(statusCode))
	fr := Frame{
		Opcode:  CloseOpcode,
		Payload: payload,
	}
	if err := c.Send(fr); err != nil {
		return err
	}

	return c.conn.Close()
}
