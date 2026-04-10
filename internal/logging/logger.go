package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct {
	base *log.Logger
}

func New() *Logger {
	return &Logger{
		base: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) write(level string, msg string, fields Field) {
	if l == nil || l.base == nil {
		return
	}

	payload := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339),
		"level": level,
		"msg":   msg,
	}

	for k, v := range fields {
		payload[k] = v
	}

	data, err := json.Marshal(payload)
	if err != nil {
		l.base.Printf(`{"ts":"%s","level":"error","msg":"log marshal failed"}`, time.Now().UTC().Format(time.RFC3339))
		return
	}

	l.base.Println(string(data))
}

func (l *Logger) Info(msg string, fields Field)  { l.write("info", msg, fields) }
func (l *Logger) Warn(msg string, fields Field)  { l.write("warn", msg, fields) }
func (l *Logger) Error(msg string, fields Field) { l.write("error", msg, fields) }
