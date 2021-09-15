package websocket

import (
	"bufio"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"
	"unicode/utf8"
)

// Close codes defined in RFC 6455.
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

func isValidReceivedCloseCode(code int) bool {
	return validReceivedCloseCodes[code] || (code >= 3000 && code <= 4999)
}

// Conn is a type which represents the WebSocket connection.
type Conn struct {
	conn net.Conn
	rw   *bufio.ReadWriter

	reader io.Reader
	writer io.WriteCloser

	closeErr *CloseError
	isServer bool
}

// NextReader returns the message type of the first fragmented frame
// (either TextOpcode or BinaryOpcode) and reader, using which you can receive
// other frame bytes.
//
// It discards the previous reader if it's not empty. There can be at most one
// open reader on a connection.
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

// ReadMessage is a helper method for getting all fragmented frames in one message.
// It uses NextReader under the hood.
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

	var maskKey []byte
	if fr.isMasked {
		maskKey, err = c.read(4)
		if err != nil {
			return fr, err
		}
	}

	payload, err := c.read(length)
	if err != nil {
		return fr, err
	}

	if fr.isMasked {
		for i := uint64(0); i < uint64(len(payload)); i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	fr.payload = payload
	if closeErr := c.validate(fr); closeErr != nil {
		c.closeErr = closeErr

		return fr, closeErr
	}

	return fr, c.processReceivedFrame(fr)
}

func (c *Conn) processReceivedFrame(fr frame) error {
	switch fr.opcode {
	case CloseOpcode:
		closeCode := CloseNormalClosure
		if len(fr.payload) >= 2 {
			closeCode = int(binary.BigEndian.Uint16(fr.payload[:2]))
		}

		if err := c.close(closeCode); err != nil {
			return err
		}

		c.closeErr = &CloseError{code: closeCode}

		return c.closeErr
	case PingOpcode:
		pongFr := fr
		pongFr.opcode = PongOpcode

		if err := c.send(pongFr); err != nil {
			return err
		}
	case ContinuationOpcode:
		if c.reader == nil {
			return c.setCloseError(errEmptyContinueFrames)
		}
	case TextOpcode, BinaryOpcode:
		if c.reader != nil {
			return c.setCloseError(errInvalidContinuationFrame)
		}
	}

	return nil
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

	if fr.IsClose() && len(fr.payload) != 0 {
		if len(fr.payload) < 2 {
			return errInvalidClosurePayload
		}

		code := int(binary.BigEndian.Uint16(fr.payload[:2]))
		reason := fr.payload[2:]

		if !isValidReceivedCloseCode(code) {
			return errInvalidClosureCode
		}

		if !utf8.Valid(reason) {
			return errInvalidUtf8Payload
		}
	}

	return nil
}

// NextWriter returns a writer using which you can send message partially.
// The writer's Close method flushes the complete message to the network.
//
// It discards the previous writer if it's not empty. There can be at most one
// open writer on a connection.
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

// WriteMessage is a helper method to send message entire.
// It uses a NextWriter under the hood.
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

	if !c.isServer {
		data[1] |= 0x80

		maskKey := newMaskKey()
		for i := 0; i < len(fr.payload); i++ {
			fr.payload[i] ^= maskKey[i%len(maskKey)]
		}

		data = append(data, maskKey[:]...)
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

// Close sends control normal close frame if wasn't any errors. After that
// the tcp connection will be closed. Otherwise, it sends close frame
// with status code depending on happened error.
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
