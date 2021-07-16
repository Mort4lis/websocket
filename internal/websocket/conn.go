package websocket

import (
	"bufio"
	"encoding/binary"
	"io"
	"io/ioutil"
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
	conn net.Conn
	rw   *bufio.ReadWriter

	reader io.Reader
	writer io.WriteCloser

	closeErr *CloseError
}

func (c *Conn) NextReader() (frameType byte, r io.Reader, err error) {
	if c.reader != nil {
		_, err = ioutil.ReadAll(c.reader)
		if err != nil {
			return noFrame, nil, err
		}

		c.reader = nil
	}

	for c.closeErr == nil {
		fr, err := c.receive()
		if err != nil {
			return noFrame, nil, err
		}

		if fr.IsText() || fr.IsBinary() {
			c.reader = newMessageReader(c, fr.opcode, fr.payload, !fr.isFragment)
			return fr.opcode, c.reader, nil
		}
	}

	return noFrame, nil, c.closeErr
}

func (c *Conn) ReadMessage() (messageType byte, payload []byte, err error) {
	frameType, r, err := c.NextReader()
	if err != nil {
		return noFrame, nil, err
	}

	payload, err = ioutil.ReadAll(r)
	if err != nil {
		return noFrame, nil, err
	}

	return frameType, payload, nil
}

func (c *Conn) receive() (frame, error) {
	fr := frame{}

	head, err := c.read(2)
	if err != nil {
		return fr, err
	}

	fr.isFragment = (head[0] & 0x80) == 0x00
	fr.reserved = head[0] & 0x70
	fr.opcode = head[0] & 0x0F
	fr.isMasked = (head[1] & 0x80) == 0x80

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
	fr.length = length

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
	fr.payload = payload

	closeErr := c.validate(fr)
	if closeErr != nil {
		c.closeErr = closeErr
		return fr, closeErr
	}

	switch fr.opcode {
	case CloseOpcode:
		closeCode := CloseNormalClosure
		if len(fr.payload) >= 2 {
			closeCode = int(binary.BigEndian.Uint16(fr.payload[:2]))
		}

		err = c.close(closeCode)
		if err != nil {
			return fr, err
		}

		c.closeErr = &CloseError{code: closeCode}
		return fr, c.closeErr
	case PingOpcode:
		pongFr := fr
		pongFr.opcode = PongOpcode
		err = c.send(pongFr)
		if err != nil {
			return fr, err
		}
	case ContinuationOpcode:
		if c.reader == nil {
			return fr, c.setCloseError(errEmptyContinueFrames)
		}
	case TextOpcode, BinaryOpcode:
		if c.reader != nil {
			return fr, c.setCloseError(errInvalidContinuationFrame)
		}
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

func (c *Conn) validate(fr frame) *CloseError {
	if fr.IsControl() && (fr.length > 125 || fr.isFragment) {
		return errInvalidControlFrame
	}
	if fr.reserved > 0 {
		return errNonZeroRSVFrame
	}
	if fr.opcode > BinaryOpcode && fr.opcode < CloseOpcode || fr.opcode > PongOpcode {
		return errReservedOpcodeFrame
	}
	if fr.IsClose() {
		if len(fr.payload) >= 2 {
			code := int(binary.BigEndian.Uint16(fr.payload[:2]))
			reason := fr.payload[2:]
			if !IsValidReceivedCloseCode(code) {
				return errInvalidClosureCode
			}
			if !utf8.Valid(reason) {
				return errInvalidUtf8Payload
			}
		} else if len(fr.payload) != 0 {
			return errInvalidClosurePayload
		}
	}

	return nil
}

func (c *Conn) NextWriter(messageType byte) (io.WriteCloser, error) {
	if c.closeErr != nil {
		return nil, c.closeErr
	}

	if c.writer != nil {
		err := c.writer.Close()
		if err != nil {
			return nil, err
		}

		c.writer = nil
	}

	c.writer = newMessageWriter(c, messageType)
	return c.writer, nil
}

func (c *Conn) WriteMessage(messageType byte, payload []byte) error {
	w, err := c.NextWriter(messageType)
	if err != nil {
		return err
	}

	_, err = w.Write(payload)
	if err != nil {
		return err
	}

	return w.Close()
}

func (c *Conn) send(fr frame) error {
	data := make([]byte, 2)

	data[0] = fr.opcode
	if !fr.isFragment {
		data[0] |= 0x80
	}

	length := uint64(len(fr.payload))
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

	data = append(data, fr.payload...)
	return c.write(data)
}

func (c *Conn) write(data []byte) error {
	if _, err := c.rw.Write(data); err != nil {
		return err
	}
	return c.rw.Flush()
}

func (c *Conn) setCloseError(err *CloseError) error {
	c.closeErr = err
	return err
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
	fr := frame{
		opcode:  CloseOpcode,
		payload: payload,
	}
	if err := c.send(fr); err != nil {
		return err
	}

	return c.conn.Close()
}
