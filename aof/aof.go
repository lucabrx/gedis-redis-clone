package aof

import (
	"bufio"
	"io"
	"os"
	"sync"
	"time"

	"github.com/lucabrx/gedis/resp"
)

type Aof struct {
	file *os.File
	rd   *bufio.Reader // reads commands from the AOF file
	mu   sync.Mutex
}

func NewAof(path string) (*Aof, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666) // open the AOF file for reading and writing
	if err != nil {
		return nil, err
	}

	aof := &Aof{
		file: f,
		rd:   bufio.NewReader(f),
	}

	// Start a sync loop to flush to disk every second
	go func() {
		for {
			aof.mu.Lock()
			aof.file.Sync()
			aof.mu.Unlock()
			time.Sleep(time.Second)
		}
	}()

	return aof, nil
}

func (aof *Aof) Close() error {
	aof.mu.Lock()
	defer aof.mu.Unlock()
	return aof.file.Close()
}

func (aof *Aof) Write(v resp.Value) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	writer := resp.NewWriter(aof.file)
	return writer.Write(v)
}

func (aof *Aof) Read(fn func(value resp.Value)) error {
	aof.mu.Lock()
	defer aof.mu.Unlock()

	// Reset file pointer to the beginning
	_, err := aof.file.Seek(0, 0)
	if err != nil {
		return err
	}

	reader := resp.NewReader(aof.file)

	for {
		value, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		fn(value)
	}

	return nil
}
