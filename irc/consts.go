package irc

// Events
const (
	TRANSFER_START       = iota // 0
	TRANSFER_PAUSE              // 1
	TRANSFER_RESUME             // 2
	TRANSFER_CANCEL             // 3
	TRANSFER_FINISH             // 4
	TRANSFER_PROGRESS           // 5
	TRANSFER_ERROR              // 6
	TRANFER_PRECOMPLETED        // 7
	TRANFER_FILE_CLOSED         // 8
)

// States
const (
	TRANSFER_STATUS_IDLE     = iota // 0
	TRANSFER_STATUS_STARTED         // 1
	TRANSFER_STATUS_PAUSED          // 2
	TRANSFER_STATUS_CANCELED        // 3
	TRANSFER_STATUS_FINISHED        // 4
	TRANSFER_STATUS_ERROR           // 5
)

const cRed = "\033[31m"
const cBlue = "\033[34m"

// 50MB buffer | Low values create too many write calls | High values create too much memory usage
const bufferSize = 1024 * 1024 * 50
