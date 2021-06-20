package websocket

const (
	TextOpcode   = 0x01
	BinaryOpcode = 0x02
	CloseOpcode  = 0x08
	PingOpcode   = 0x09
	PongOpcode   = 0xA
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

func (f Frame) IsControl() bool {
	return f.Opcode == CloseOpcode || f.Opcode == PingOpcode || f.Opcode == PongOpcode
}

func (f Frame) validate() error {
	if f.IsControl() && (f.Length > maxInt8Value-2 || f.IsFragment) {
		return ControlFrameErr
	}
	if f.Reserved > 0 {
		return NonZeroRSVFrameErr
	}

	return nil
}
