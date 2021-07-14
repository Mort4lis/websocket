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

type Frame struct {
	IsFragment bool
	Reserved   byte
	Opcode     byte
	IsMasked   bool
	Length     uint64
	Payload    []byte
}

func (f Frame) IsText() bool {
	return f.Opcode == TextOpcode
}

func (f Frame) IsBinary() bool {
	return f.Opcode == BinaryOpcode
}

func (f Frame) IsContinuation() bool {
	return f.Opcode == ContinuationOpcode
}

func (f Frame) IsClose() bool {
	return f.Opcode == CloseOpcode
}

func (f Frame) IsControl() bool {
	return f.Opcode == CloseOpcode || f.Opcode == PingOpcode || f.Opcode == PongOpcode
}
