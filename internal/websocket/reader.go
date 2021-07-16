package websocket

import (
	"io"
	"unicode/utf8"
)

type messageReader struct {
	conn   *Conn
	typ    byte
	isLast bool
	pos    int
	buff   []byte
}

func (r *messageReader) Read(p []byte) (int, error) {
	if r.isEOF() {
		return 0, io.EOF
	}

	if !r.isLast && len(r.buff[r.pos:]) < len(p) {
		var fr Frame
		var err error

		for r.conn.closeErr == nil {
			fr, err = r.conn.receive()
			if err != nil {
				return 0, err
			}

			if !fr.IsControl() {
				break
			}
		}

		r.isLast = !fr.IsFragment
		r.buff = append(r.buff, fr.Payload...)
	}

	if r.isLast && r.typ == TextOpcode && !utf8.Valid(r.buff) {
		return 0, r.conn.setCloseError(errInvalidUtf8Payload)
	}

	n := copy(p, r.buff[r.pos:])
	r.pos += n
	return n, nil
}

func (r *messageReader) isEOF() bool {
	return r.conn.closeErr != nil || r.conn.reader != r || (r.isLast && len(r.buff[r.pos:]) == 0)
}
