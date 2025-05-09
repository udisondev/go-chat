package pack

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func ReadFrom(r io.Reader, buf []byte) (int, error) {
	var packl uint16
	err := binary.Read(r, binary.LittleEndian, &packl)
	if err != nil {
		return 0, fmt.Errorf("read pack len: %w", err)
	}

	l := int(packl)

	if l > len(buf) {
		return 0, errors.New("pack too big")
	}

	for read := 0; read < l; {
		n, err := r.Read(buf[read:])
		if err != nil {
			return read, fmt.Errorf("read pack: %w", err)
		}
		read += n
	}

	return l, nil
}

func WriteTo(w io.Writer, b []byte) (int, error) {
	err := binary.Write(w, binary.LittleEndian, uint16(len(b)))
	if err != nil {
		return 0, fmt.Errorf("write pack len: %w", err)
	}
	for written := 0; written < len(b); {
		n, err := w.Write(b[written:])
		if err != nil {
			return written, fmt.Errorf("write pack: %w", err)
		}
		written += n
	}
	return len(b), nil
}
