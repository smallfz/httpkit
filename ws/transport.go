package ws

import (
	"io"
	"net"
	"sync"
)

const MAX_BUF_SIZE = 1024 * 1024 * 512

const BUF_SIZE_EXTEND = 1024 * 8

type Transport interface {
	io.ReadWriteCloser
	RemoteAddresser
}

type wsTransport struct {
	conn          WSConn
	readingBuf    []byte
	readingBufLen int
	lck           *sync.Mutex
	lckRead       *sync.Mutex
	frag          *WSFrame
}

var _ RemoteAddresser = (*wsTransport)(nil)

func (t *wsTransport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *wsTransport) Close() error {
	return t.conn.Close()
}

func (t *wsTransport) Write(data []byte) (int, error) {
	f := &WSFrame{
		Fin:  true,
		Op:   2, // binary frame.
		Data: data,
	}
	return t.conn.WriteFrame(f)
}

func (t *wsTransport) readFromBuf(buf []byte) (int, error) {
	t.lck.Lock()
	defer t.lck.Unlock()
	if t.readingBuf == nil {
		t.readingBuf = make([]byte, 1024*4)
	}
	if t.readingBufLen > 0 {
		size := len(buf)
		if size > t.readingBufLen {
			size = t.readingBufLen
		}
		copy(buf[:size], t.readingBuf[:size])
		restLen := t.readingBufLen - size
		copy(t.readingBuf[:restLen], t.readingBuf[size:t.readingBufLen])
		t.readingBufLen = restLen
		return size, nil
	}
	return 0, nil
}

func (t *wsTransport) appendToBuf(dat []byte) error {
	if len(dat) == 0 {
		return nil
	}
	t.lck.Lock()
	defer t.lck.Unlock()
	if t.readingBuf == nil {
		t.readingBuf = make([]byte, 1024*4)
	}
	sizeAfter := t.readingBufLen + len(dat)
	if sizeAfter > cap(t.readingBuf) {
		sizeAfterExtended := len(t.readingBuf)
		for {
			sizeAfterExtended += BUF_SIZE_EXTEND
			if sizeAfterExtended > MAX_BUF_SIZE {
				return io.ErrShortBuffer
			}
			if sizeAfterExtended >= sizeAfter {
				bufNew := make([]byte, sizeAfterExtended)
				copy(bufNew, t.readingBuf[:t.readingBufLen])
				t.readingBuf = bufNew
				break
			}
		}
	}
	copy(t.readingBuf[t.readingBufLen:t.readingBufLen+len(dat)], dat)
	t.readingBufLen += len(dat)
	return nil
}

func (t *wsTransport) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	if size, err := t.readFromBuf(buf); err == nil && size > 0 {
		return size, nil
	}

	t.lckRead.Lock()
	defer t.lckRead.Unlock()

	for {
		f, err := t.conn.ReadFrame()
		if err != nil {
			return 0, err
		}

		if f.Op != 2 && f.Op != 0 {
			// slog.Debug("frame ignored:",
			// 	"op", f.Op, "fin", f.Fin, "len", len(f.Data),
			// )
			continue
		}

		if len(f.Data) <= 0 {
			continue
		}

		if err := t.appendToBuf(f.Data); err != nil {
			return 0, err
		}
		if size, err := t.readFromBuf(buf); err == nil && size > 0 {
			return size, nil
		}
	}
}

func MakeTransport(conn WSConn) Transport {
	return &wsTransport{
		conn:    conn,
		lck:     new(sync.Mutex),
		lckRead: new(sync.Mutex),
	}
}
