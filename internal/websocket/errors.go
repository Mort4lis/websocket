package websocket

import "fmt"

type CloseError struct {
	code int
	text string
}

func NewCloseError(code int, text string) *CloseError {
	return &CloseError{code, text}
}

func IsCloseError(err error) bool {
	_, ok := err.(*CloseError)
	return ok
}

func (e *CloseError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.text)
}

var (
	errInvalidControlFrame = NewCloseError(
		CloseProtocolError,
		"all control frames must have a payload length of 125 bytes or less and must not be fragmented",
	)
	errNonZeroRSVFrame = NewCloseError(
		CloseProtocolError,
		"reserved bits must be set at 0, when no extension defining RSV meaning has been negotiated",
	)
	errReservedOpcodeFrame = NewCloseError(
		CloseProtocolError,
		"opcodes 0x03-0x07 and 0xB-0xF are reserved for further frames",
	)
	errInvalidContinuationFrame = NewCloseError(
		CloseProtocolError,
		"the fragmented frame after initial frame doesn't have continuation opcode",
	)
	errEmptyContinueFrames = NewCloseError(
		CloseProtocolError,
		"there is no frames to continue",
	)
	errInvalidClosurePayload = NewCloseError(
		CloseProtocolError,
		"invalid close payload",
	)
	errInvalidClosureCode = NewCloseError(
		CloseProtocolError,
		"invalid closure code",
	)
	errInvalidUtf8Payload = NewCloseError(
		CloseInvalidFramePayloadData,
		"invalid UTF-8 text payload",
	)
)
