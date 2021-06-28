package websocket

import "fmt"

type CloseError struct {
	code int
	text string
}

func NewCloseError(code int, text string) CloseError {
	return CloseError{code, text}
}

func IsCloseError(err error) bool {
	_, ok := err.(CloseError)
	return ok
}

func (e CloseError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.text)
}

var (
	errInvalidControlFrame = NewCloseError(
		protocolError,
		"all control frames must have a payload length of 125 bytes or less and must not be fragmented",
	)
	errNonZeroRSVFrame = NewCloseError(
		protocolError,
		"reserved bits must be set at 0, when no extension defining RSV meaning has been negotiated",
	)
	errReservedOpcodeFrame = NewCloseError(
		protocolError,
		"opcodes 0x03-0x07 and 0xB-0xF are reserved for further frames",
	)
	errInvalidContinuationFrame = NewCloseError(
		protocolError,
		"the fragmented frame after initial frame doesn't have continuation opcode",
	)
	errEmptyContinueFrames = NewCloseError(
		protocolError,
		"there is no frames to continue",
	)
	errInvalidUtf8Payload = NewCloseError(
		invalidPayload,
		"invalid UTF-8 text payload",
	)
)
