package logger

import (
	"fmt"
	"sync"
	"time"
)

var (
	mu   sync.Mutex
	logs []string
)

func Log(msg string) {
	mu.Lock()
	defer mu.Unlock()
	logLine := fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), msg)
	logs = append(logs, logLine)
}

func GetLogs() []string {
	mu.Lock()
	defer mu.Unlock()
	out := make([]string, len(logs))
	copy(out, logs)
	return out
}
