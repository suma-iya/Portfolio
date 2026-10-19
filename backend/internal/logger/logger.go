// Package logger provides a minimal structured JSON logger.
//
// log/slog (stdlib, Go 1.21+) would be the natural choice here, but this
// module targets Go 1.20 for broad compatibility with older toolchains
// (e.g. distro-packaged Go on the deploy target), so we roll a small
// equivalent instead.
package logger

import (
	"encoding/json"
	"os"
	"time"
)

type Logger struct {
	out *os.File
}

func New() *Logger {
	return &Logger{out: os.Stdout}
}

func (l *Logger) write(level, msg string, kv ...interface{}) {
	fields := map[string]interface{}{
		"time":  time.Now().UTC().Format(time.RFC3339),
		"level": level,
		"msg":   msg,
	}
	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		fields[key] = kv[i+1]
	}
	line, err := json.Marshal(fields)
	if err != nil {
		return
	}
	l.out.Write(append(line, '\n'))
}

func (l *Logger) Info(msg string, kv ...interface{})  { l.write("info", msg, kv...) }
func (l *Logger) Error(msg string, kv ...interface{}) { l.write("error", msg, kv...) }
