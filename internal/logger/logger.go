package logger

import (
	"bufio"
	"os"
	"path/filepath"
	"sync"
)

type Logger struct {
	mu sync.Mutex
	w  *bufio.Writer
	f  *os.File
}

func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{f: f, w: bufio.NewWriterSize(f, 256*1024)}, nil
}

func (l *Logger) WriteLine(data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.w.Write(data)
	l.w.WriteByte('\n')
	l.w.Flush()
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.w.Flush()
	return l.f.Close()
}
