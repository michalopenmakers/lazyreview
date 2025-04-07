package notifications

import (
	"fmt"
	"time"
)

func SendNotification(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Println("Notification at", timestamp+":", message)
}
