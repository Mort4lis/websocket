package websocket

const (
	noFrame            = 0xff
	ContinuationOpcode = 0x00
	TextOpcode         = 0x01
	BinaryOpcode       = 0x02
	CloseOpcode        = 0x08
	PingOpcode         = 0x09
	PongOpcode         = 0xA
)

type frame struct {
	isFragment bool
	reserved   byte
	opcode     byte
	isMasked   bool
	length     uint64
	payload    []byte
}

func (f frame) IsText() bool {
	return f.opcode == TextOpcode
}

func (f frame) IsBinary() bool {
	return f.opcode == BinaryOpcode
}

func (f frame) IsContinuation() bool {
	return f.opcode == ContinuationOpcode
}

func (f frame) IsClose() bool {
	return f.opcode == CloseOpcode
}

func (f frame) IsControl() bool {
	return f.opcode == CloseOpcode || f.opcode == PingOpcode || f.opcode == PongOpcode
}
