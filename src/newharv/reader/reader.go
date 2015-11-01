package reader

import (
	"bufio"
	"golang.org/x/net/context"
	"io"
)

type LineReader struct {
	buf *bufio.Reader
	rd  io.Closer
}

func NewLineReader(rd io.ReadCloser) LineReader {
	return LineReader{bufio.NewReader(rd), rd}
}

func (lr LineReader) Next(ctx context.Context) (line string, err error) {
	c := make(chan struct{})
	go func() {
		line, err = lr.next()
		c <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		lr.rd.Close()
		return "", ctx.Err()
	case <-c:
		return line, err
	}
}

func (lr LineReader) next() (string, error) {
	skip := false
	for {
		line, isPrefix, err := lr.buf.ReadLine()
		if err != nil {
			return "", err
		}

		// skip to long line
		if isPrefix || skip {
			skip = isPrefix
			continue
		}

		return string(line), nil
	}
}
