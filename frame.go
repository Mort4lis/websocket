package websocket

// Type of frames which defines in RFC 6455.
const (
	ContinuationOpcode = 0x00
	TextOpcode         = 0x01
	BinaryOpcode       = 0x02
	CloseOpcode        = 0x08
	PingOpcode         = 0x09
	PongOpcode         = 0xA

	noFrame = 0xff
)

type frame struct {
	isFragment bool
	reserved   byte
	opcode     byte
	isMasked   bool
	length     uint64
	payload    []byte
}

func (f frame) isText() bool {
	return f.opcode == TextOpcode
}

func (f frame) isBinary() bool {
	return f.opcode == BinaryOpcode
}

func (f frame) isContinuation() bool {
	return f.opcode == ContinuationOpcode
}

func (f frame) isClose() bool {
	return f.opcode == CloseOpcode
}

func (f frame) isControl() bool {
	return f.opcode == CloseOpcode || f.opcode == PingOpcode || f.opcode == PongOpcode
}
