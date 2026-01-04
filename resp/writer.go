package resp

import (
	"io"
	"strconv"
	"sync"
)

type Writer struct {
	writer io.Writer
	mu     sync.Mutex
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: w}
}

func (w *Writer) Write(v Value) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.write(v)
}

func (w *Writer) write(v Value) error {
	var bytes []byte
	switch v.Type {
	case "string":
		bytes = []byte(string(STRING) + v.Str + "\r\n")
	case "error":
		bytes = []byte(string(ERROR) + v.Str + "\r\n")
	case "integer":
		bytes = []byte(string(INTEGER) + strconv.Itoa(v.Num) + "\r\n")
	case "null":
		bytes = []byte("$-1\r\n")
	case "bulk":
		bytes = []byte(string(BULK) + strconv.Itoa(len(v.Bulk)) + "\r\n" + v.Bulk + "\r\n")
	case "array":
		bytes = []byte(string(ARRAY) + strconv.Itoa(len(v.Array)) + "\r\n")
		if _, err := w.writer.Write(bytes); err != nil {
			return err
		}
		for _, val := range v.Array {
			// Recursive call must use write, not Write, to avoid deadlock!
			if err := w.write(val); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}

	_, err := w.writer.Write(bytes)
	return err
}
