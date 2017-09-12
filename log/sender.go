package log

// Sender contains functions that allow messages to be sent from the server
// to a client
type Sender interface {
	// SendMessage sends a message
	SendMessage(lvl Level, message string)
}
