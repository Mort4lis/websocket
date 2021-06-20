package websocket

import "fmt"

type Error struct {
	statusCode  uint16
	description string
}

func NewError(statusCode uint16, description string) Error {
	return Error{statusCode, description}
}

func (e Error) Error() string {
	return fmt.Sprintf("%d: %s", e.statusCode, e.description)
}

var (
	ControlFrameErr = NewError(
		protocolError,
		"all control frames must have a payload length of 125 bytes or less and must not be fragmented",
	)
	NonZeroRSVFrameErr = NewError(
		protocolError,
		"reserved bits must be set at 0, when no extension defining RSV meaning has been negotiated",
	)
)
