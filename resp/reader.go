package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

const (
	STRING  = '+'
	ERROR   = '-'
	INTEGER = ':'
	BULK    = '$'
	ARRAY   = '*'
)

type Value struct {
	Type  string
	Str   string
	Num   int
	Bulk  string
	Array []Value
}

type Reader struct {
	r *bufio.Reader
}

func NewReader(rd io.Reader) *Reader {
	return &Reader{r: bufio.NewReader(rd)}
}

func (r *Reader) Read() (Value, error) {
	_type, err := r.r.ReadByte()
	if err != nil {
		return Value{}, err
	}

	switch _type {
	case ARRAY:
		return r.readArray()
	case BULK:
		return r.readBulk()
	case STRING:
		return r.readSimpleString()
	case ERROR:
		v, err := r.readSimpleString()
		v.Type = "error"
		return v, err
	case INTEGER:
		return r.readInteger()
	default:
		fmt.Printf("Unknown type: %v\n", string(_type))
		return Value{}, fmt.Errorf("unknown type: %v", string(_type))
	}
}

func (r *Reader) readLine() (line []byte, n int, err error) {
	for {
		b, err := r.r.ReadByte()
		if err != nil {
			return nil, 0, err
		}
		n += 1
		line = append(line, b)
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}
	return line[:len(line)-2], n, nil
}

func (r *Reader) readInteger() (Value, error) {
	line, _, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	i64, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return Value{}, err
	}
	return Value{Type: "integer", Num: int(i64)}, nil
}

func (r *Reader) readArray() (Value, error) {
	v := Value{Type: "array"}

	line, _, err := r.readLine()
	if err != nil {
		return v, err
	}

	len, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return v, err
	}

	v.Array = make([]Value, 0, len)
	for i := 0; i < int(len); i++ {
		val, err := r.Read()
		if err != nil {
			return v, err
		}
		v.Array = append(v.Array, val)
	}

	return v, nil
}

func (r *Reader) readBulk() (Value, error) {
	v := Value{Type: "bulk"}

	line, _, err := r.readLine()
	if err != nil {
		return v, err
	}

	len, err := strconv.ParseInt(string(line), 10, 64)
	if err != nil {
		return v, err
	}

	if len == -1 {
		return Value{Type: "null"}, nil
	}

	bulk := make([]byte, len)
	_, err = io.ReadFull(r.r, bulk)
	if err != nil {
		return v, err
	}

	v.Bulk = string(bulk)

	_, _, err = r.readLine()
	if err != nil {
		return v, err
	}

	return v, nil
}

func (r *Reader) readSimpleString() (Value, error) {
	line, _, err := r.readLine()
	if err != nil {
		return Value{}, err
	}
	return Value{Type: "string", Str: string(line)}, nil
}
