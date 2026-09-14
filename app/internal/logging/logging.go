package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu   sync.Mutex
	file *os.File
	path string
)

func Init(dir string) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	p := filepath.Join(dir, "airborne.log")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Close()
	}
	file = f
	path = p
}

func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return path
}

func write(level, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	fmt.Fprintf(file, "[%s] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), level, fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) {
	write("INFO ", format, args...)
}

func Errorf(format string, args ...any) {
	write("ERROR", format, args...)
}

func Block(title, content string) {
	mu.Lock()
	defer mu.Unlock()
	if file == nil {
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(file, "[%s] ---- %s ----\n%s\n---- /%s ----\n", ts, title, content, title)
}
