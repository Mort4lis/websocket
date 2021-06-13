package websocket

import "encoding/binary"

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

	for i := 0; i < len(payload); i++ {
		payload[i] ^= maskKey[i%4]
	}
	frame.Payload = payload

	return frame, nil
}

func (ws *Websocket) read(size uint64) ([]byte, error) {
	buff := make([]byte, size)
	if _, err := ws.buff.Read(buff); err != nil {
		return nil, err
	}
	return buff, nil
}
