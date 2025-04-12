package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

type memoryHandler struct {
	mu      sync.Mutex
	logs    []string
	handler slog.Handler
}

func (m *memoryHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return m.handler.Enabled(ctx, level)
}

func (m *memoryHandler) Handle(ctx context.Context, r slog.Record) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	formatted := fmt.Sprintf("[%s] %s", r.Time.Format("2006-01-02 15:04:05"), r.Message)
	m.logs = append(m.logs, formatted)
	return m.handler.Handle(ctx, r)
}

func (m *memoryHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &memoryHandler{
		handler: m.handler.WithAttrs(attrs),
		logs:    m.logs,
	}
}

func (m *memoryHandler) WithGroup(name string) slog.Handler {
	return &memoryHandler{
		handler: m.handler.WithGroup(name),
		logs:    m.logs,
	}
}

var memHandler *memoryHandler
var Logger *slog.Logger

func init() {
	memHandler = &memoryHandler{
		handler: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
		}),
	}
	Logger = slog.New(memHandler)
}

func Log(msg string) {
	Logger.Info(msg)
}

func GetLogs() []string {
	memHandler.mu.Lock()
	defer memHandler.mu.Unlock()
	copied := make([]string, len(memHandler.logs))
	copy(copied, memHandler.logs)
	return copied
}
