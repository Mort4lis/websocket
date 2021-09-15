package websocket

import "fmt"

// HandshakeError is a type which represents an error occurs
// in process handshake to establish WebSocket connection.
type HandshakeError struct {
	reason string
}

func (e HandshakeError) Error() string {
	return e.reason
}

// CloseError is a type which represents closure WebSocket error.
type CloseError struct {
	code int
	text string
}

func newCloseError(code int, text string) *CloseError {
	return &CloseError{code, text}
}

func (e *CloseError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.text)
}

var (
	errInvalidControlFrame = newCloseError(
		CloseProtocolError,
		"all control frames must have a payload length of 125 bytes or less and must not be fragmented",
	)
	errNonZeroRSVFrame = newCloseError(
		CloseProtocolError,
		"reserved bits must be set at 0, when no extension defining RSV meaning has been negotiated",
	)
	errReservedOpcodeFrame = newCloseError(
		CloseProtocolError,
		"opcodes 0x03-0x07 and 0xB-0xF are reserved for further frames",
	)
	errInvalidContinuationFrame = newCloseError(
		CloseProtocolError,
		"the fragmented frame after initial frame doesn't have continuation opcode",
	)
	errEmptyContinueFrames = newCloseError(
		CloseProtocolError,
		"there is no frames to continue",
	)
	errInvalidClosurePayload = newCloseError(
		CloseProtocolError,
		"invalid close payload",
	)
	errInvalidClosureCode = newCloseError(
		CloseProtocolError,
		"invalid closure code",
	)
	errInvalidUtf8Payload = newCloseError(
		CloseInvalidFramePayloadData,
		"invalid UTF-8 text payload",
	)
)
