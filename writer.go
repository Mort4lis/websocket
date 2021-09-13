package websocket

const defaultWriteBufferSize = 4096

type messageWriter struct {
	conn        *Conn
	messageType byte
	wasFragment bool

	pos  int
	buff []byte
}

func newMessageWriter(conn *Conn, frameType byte) *messageWriter {
	return &messageWriter{
		conn:        conn,
		messageType: frameType,
		buff:        make([]byte, defaultWriteBufferSize),
	}
}

func (w *messageWriter) Write(p []byte) (int, error) {
	if w.conn.closeErr != nil {
		return 0, w.conn.closeErr
	}

	n := 0

	for len(p) > 0 {
		if len(w.buff[w.pos:]) == 0 {
			fr := frame{
				isFragment: true,
				opcode:     w.getOpcode(),
				payload:    w.buff,
			}

			err := w.conn.send(fr)
			if err != nil {
				return 0, err
			}

			w.pos = 0
			w.wasFragment = true
		}

		nn := copy(w.buff[w.pos:], p)
		p = p[nn:]

		n += nn
		w.pos += nn
	}

	return n, nil
}

func (w *messageWriter) getOpcode() byte {
	if w.wasFragment {
		return ContinuationOpcode
	}

	return w.messageType
}

func (w *messageWriter) Close() error {
	if w.conn.closeErr != nil {
		return w.conn.closeErr
	}

	fr := frame{
		opcode:  w.getOpcode(),
		payload: w.buff[:w.pos],
	}

	if err := w.conn.send(fr); err != nil {
		return err
	}

	return nil
}
