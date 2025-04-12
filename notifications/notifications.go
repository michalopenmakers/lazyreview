package notifications

import (
	"time"
)

func SendNotification(message string) {
	_ = time.Now().Format("2006-01-02 15:04:05")
}
