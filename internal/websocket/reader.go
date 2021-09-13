package websocket

import (
	"io"
	"unicode/utf8"
)

type messageReader struct {
	conn        *Conn
	messageType byte
	isLast      bool
	pos         int
	buff        []byte
}

func newMessageReader(conn *Conn, messageType byte, buff []byte, isLast bool) *messageReader {
	return &messageReader{
		conn:        conn,
		messageType: messageType,
		isLast:      isLast,
		buff:        buff,
	}
}

func (r *messageReader) Read(p []byte) (int, error) {
	if r.isEOF() {
		return 0, io.EOF
	}

	if !r.isLast && len(r.buff[r.pos:]) < len(p) {
		var (
			fr  frame
			err error
		)

		for r.conn.closeErr == nil {
			fr, err = r.conn.receive()
			if err != nil {
				return 0, err
			}

			if !fr.IsControl() {
				break
			}
		}

		r.isLast = !fr.isFragment
		r.buff = append(r.buff, fr.payload...)
	}

	if r.isLast && r.messageType == TextOpcode && !utf8.Valid(r.buff) {
		return 0, r.conn.setCloseError(errInvalidUtf8Payload)
	}

	n := copy(p, r.buff[r.pos:])
	r.pos += n

	return n, nil
}

func (r *messageReader) isEOF() bool {
	return r.conn.closeErr != nil || r.conn.reader != r || (r.isLast && len(r.buff[r.pos:]) == 0)
}
