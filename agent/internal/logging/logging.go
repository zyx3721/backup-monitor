// Package logging 提供按大小轮转的文件日志。
package logging

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	DefaultMaxMb    = 10
	DefaultBackups  = 2
	defaultFileMode = 0o644
	utf8BOM         = "\xEF\xBB\xBF"
)

// Setup 打开轮转日志文件并返回 logger 与关闭函数，路径为空时退回 stderr。
func Setup(path string, maxMb, backups int) (*log.Logger, func() error, error) {
	if path == "" {
		logger := log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
		return logger, func() error { return nil }, nil
	}
	if maxMb <= 0 {
		maxMb = DefaultMaxMb
	}
	if backups <= 0 {
		backups = DefaultBackups
	}

	writer, err := newRotatingWriter(path, int64(maxMb)*1024*1024, backups)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file %s: %w", path, err)
	}
	logger := log.New(writer, "", log.LstdFlags|log.Lmicroseconds)
	return logger, writer.Close, nil
}

type rotatingWriter struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

func newRotatingWriter(path string, maxSize int64, maxBackups int) (*rotatingWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	writer := &rotatingWriter{path: path, maxSize: maxSize, maxBackups: maxBackups}
	if err := writer.open(); err != nil {
		return nil, err
	}
	return writer, nil
}

func (w *rotatingWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, defaultFileMode)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	w.file = file
	w.size = info.Size()
	if runtime.GOOS == "windows" && w.size == 0 {
		if _, err := file.WriteString(utf8BOM); err != nil {
			file.Close()
			w.file = nil
			return err
		}
		w.size = int64(len(utf8BOM))
	}
	return nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, fmt.Errorf("log file %s is closed", w.path)
	}
	if w.size > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) rotate() error {
	if err := w.file.Close(); err != nil {
		return err
	}
	w.file = nil

	highest := fmt.Sprintf("%s.%d", w.path, w.maxBackups)
	if err := os.Remove(highest); err != nil && !os.IsNotExist(err) {
		return err
	}
	for index := w.maxBackups - 1; index >= 1; index-- {
		from := fmt.Sprintf("%s.%d", w.path, index)
		to := fmt.Sprintf("%s.%d", w.path, index+1)
		if err := os.Rename(from, to); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(w.path, w.path+".1"); err != nil && !os.IsNotExist(err) {
		return err
	}
	return w.open()
}

// Close 关闭底层日志文件。
func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
