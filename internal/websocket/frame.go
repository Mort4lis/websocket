package websocket

import (
	"encoding/binary"
	"io"
)

type Frame struct {
	IsFragment bool
	Reserved   byte
	Opcode     byte
	IsMasked   bool
	Length     uint64
	Payload    []byte
}

func (ws *Websocket) Receive() (Frame, error) {
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
	if length == 126 {
		lenBytes, err := ws.read(2)
		if err != nil {
			return frame, err
		}

		length = uint64(binary.BigEndian.Uint16(lenBytes))
	} else if length == 127 {
		lenBytes, err := ws.read(8)
		if err != nil {
			return frame, err
		}

		length = binary.BigEndian.Uint64(lenBytes)
	}
	frame.Length = length

	maskKey, err := ws.read(4)
	if err != nil {
		return frame, nil
	}

	payload, err := ws.read(length)
	if err != nil {
		return frame, nil
	}

	for i := uint64(0); i < uint64(len(payload)); i++ {
		payload[i] ^= maskKey[i%4]
	}
	frame.Payload = payload

	return frame, nil
}

func (ws *Websocket) read(size uint64) ([]byte, error) {
	buff := make([]byte, size)
	if _, err := io.ReadFull(ws.rw, buff); err != nil {
		return nil, err
	}

	return buff, nil
}

func (ws *Websocket) Send(frame Frame) error {
	data := make([]byte, 2)

	data[0] = frame.Opcode
	if !frame.IsFragment {
		data[0] |= 0x80
	}

	var length uint64
	if frame.Length != 0 {
		length = frame.Length
	} else {
		length = uint64(len(frame.Payload))
	}

	if length <= 125 {
		data[1] = byte(length)
	} else if length <= (1<<16)-1 {
		data[1] = 126

		lenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lenBytes, uint16(length))
		data = append(data, lenBytes...)
	} else {
		data[1] = 127

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
	frame := Frame{
		Opcode:  0x08,
		Payload: make([]byte, 2),
	}
	if err := ws.Send(frame); err != nil {
		return err
	}

	return ws.conn.Close()
}
