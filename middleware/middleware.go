package middleware

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

type Middleware func(io.ReadWriter) io.ReadWriter

type Wrapper struct {
	downstream io.ReadWriter
	io.Reader
	preparer func([]byte) ([]byte, error)
}

func (w *Wrapper) Write(b []byte) (int, error) {
	if w.preparer != nil {
		var prepErr error
		b, prepErr = w.preparer(b)
		if prepErr != nil {
			return 0, prepErr
		}
	}
	err := binary.Write(w.downstream, binary.LittleEndian, uint16(len(b)))
	if err != nil {
		return 0, fmt.Errorf("write len: %w", err)
	}
	for written := 0; written < len(b); {
		n, err := w.downstream.Write(b[written:])
		if err != nil {
			return written, fmt.Errorf("read payload: %w", err)
		}
		written += n
	}
	return len(b), nil
}

func readDownstream(
	buf []byte,
	minLen int,
	downstream io.Reader,
	upstream io.Writer,
	handler func([]byte) ([]byte, error),
) error {
	var mlen uint16
	err := binary.Read(downstream, binary.LittleEndian, &mlen)
	if err != nil {
		return fmt.Errorf("read message len: %w", err)
	}
	if int(mlen) > len(buf) {
		return errors.New("too big message")
	}
	read := 0
	for read < int(mlen) {
		n, err := downstream.Read(buf[read:])
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}
		read += n
	}
	if read < minLen {
		return errors.New("too short message")
	}
	out, err := handler(buf[:mlen])
	if err != nil {
		return err
	}
	err = binary.Write(upstream, binary.LittleEndian, uint16(len(out)))
	if err != nil {
		return err
	}
	for written := 0; written < len(out); {
		n, err := upstream.Write(out[written:])
		if err != nil {
			return fmt.Errorf("upstream message: %w", err)
		}
		written += n
	}

	return nil
}
