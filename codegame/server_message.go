package codegame

import (
	"fmt"
	"io"

	. "ai_cup_22/model"

	. "ai_cup_22/stream"
)

// Message sent from server
type ServerMessage interface {
	// Write ServerMessage to writer
	Write(writer io.Writer)

	// Get string representation of ServerMessage
	String() string
}

// Read ServerMessage from reader
func ReadServerMessage(reader io.Reader) ServerMessage {
	switch ReadInt32(reader) {
	case 0:
		return ReadServerMessageUpdateConstants(reader)
	case 1:
		return ReadServerMessageGetOrder(reader)
	case 2:
		return ReadServerMessageFinish(reader)
	case 3:
		return ReadServerMessageDebugUpdate(reader)
	}
	panic("Unexpected tag value")
}

// Update constants
type ServerMessageUpdateConstants struct {
	// New constants
	Constants Constants
}

func NewServerMessageUpdateConstants(constants Constants) ServerMessageUpdateConstants {
	return ServerMessageUpdateConstants{
		Constants: constants,
	}
}

// Read UpdateConstants from reader
func ReadServerMessageUpdateConstants(reader io.Reader) ServerMessageUpdateConstants {
	var constants Constants
	constants = ReadConstants(reader)
	return ServerMessageUpdateConstants{
		Constants: constants,
	}
}

// Write UpdateConstants to writer
func (serverMessageUpdateConstants ServerMessageUpdateConstants) Write(writer io.Writer) {
	WriteInt32(writer, 0)
	constants := serverMessageUpdateConstants.Constants
	constants.Write(writer)
}

// Get string representation of UpdateConstants
func (serverMessageUpdateConstants ServerMessageUpdateConstants) String() string {
	stringResult := "{ "
	stringResult += "Constants: "
	constants := serverMessageUpdateConstants.Constants
	stringResult += constants.String()
	stringResult += " }"
	return stringResult
}

// Get order for next tick
type ServerMessageGetOrder struct {
	// Player's view
	PlayerView Game
	// Whether app is running with debug interface available
	DebugAvailable bool
}

func NewServerMessageGetOrder(playerView Game, debugAvailable bool) ServerMessageGetOrder {
	return ServerMessageGetOrder{
		PlayerView:     playerView,
		DebugAvailable: debugAvailable,
	}
}

// Read GetOrder from reader
func ReadServerMessageGetOrder(reader io.Reader) ServerMessageGetOrder {
	var playerView Game
	playerView = ReadGame(reader)
	var debugAvailable bool
	debugAvailable = ReadBool(reader)
	return ServerMessageGetOrder{
		PlayerView:     playerView,
		DebugAvailable: debugAvailable,
	}
}

// Write GetOrder to writer
func (serverMessageGetOrder ServerMessageGetOrder) Write(writer io.Writer) {
	WriteInt32(writer, 1)
	playerView := serverMessageGetOrder.PlayerView
	playerView.Write(writer)
	debugAvailable := serverMessageGetOrder.DebugAvailable
	WriteBool(writer, debugAvailable)
}

// Get string representation of GetOrder
func (serverMessageGetOrder ServerMessageGetOrder) String() string {
	stringResult := "{ "
	stringResult += "PlayerView: "
	playerView := serverMessageGetOrder.PlayerView
	stringResult += playerView.String()
	stringResult += ", "
	stringResult += "DebugAvailable: "
	debugAvailable := serverMessageGetOrder.DebugAvailable
	stringResult += fmt.Sprint(debugAvailable)
	stringResult += " }"
	return stringResult
}

// Signifies end of the game
type ServerMessageFinish struct {
}

func NewServerMessageFinish() ServerMessageFinish {
	return ServerMessageFinish{}
}

// Read Finish from reader
func ReadServerMessageFinish(reader io.Reader) ServerMessageFinish {
	return ServerMessageFinish{}
}

// Write Finish to writer
func (serverMessageFinish ServerMessageFinish) Write(writer io.Writer) {
	WriteInt32(writer, 2)
}

// Get string representation of Finish
func (serverMessageFinish ServerMessageFinish) String() string {
	stringResult := "{ "
	stringResult += " }"
	return stringResult
}

// Debug update
type ServerMessageDebugUpdate struct {
}

func NewServerMessageDebugUpdate() ServerMessageDebugUpdate {
	return ServerMessageDebugUpdate{}
}

// Read DebugUpdate from reader
func ReadServerMessageDebugUpdate(reader io.Reader) ServerMessageDebugUpdate {
	return ServerMessageDebugUpdate{}
}

// Write DebugUpdate to writer
func (serverMessageDebugUpdate ServerMessageDebugUpdate) Write(writer io.Writer) {
	WriteInt32(writer, 3)
}

// Get string representation of DebugUpdate
func (serverMessageDebugUpdate ServerMessageDebugUpdate) String() string {
	stringResult := "{ "
	stringResult += " }"
	return stringResult
}
