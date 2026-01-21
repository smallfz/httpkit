package ws

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

type WSFrame struct {
	Fin  bool
	Op   uint8
	Data []byte
}

func NewTextFrame(text string) *WSFrame {
	return &WSFrame{
		Fin:  true,
		Op:   1,
		Data: []byte(text),
	}
}

func NewBinaryFrame(dat []byte) *WSFrame {
	return &WSFrame{
		Fin:  true,
		Op:   2,
		Data: dat,
	}
}

func WriteTextFrame(conn WSConn, text string) (int, error) {
	dat := []byte(text)

	chunkSizeMax := 1024
	offset := 0

	fragment := false
	if len(dat) > chunkSizeMax {
		fragment = true
	}

	if !fragment {
		f := &WSFrame{
			Fin:  true,
			Op:   1,
			Data: dat,
		}
		return conn.WriteFrame(f)
	}

	frames := []*WSFrame{}

	for {
		if offset >= len(dat) {
			break
		}

		fin := false
		op := byte(1)
		if offset > 0 {
			op = 0
		}

		chunkSize := chunkSizeMax
		if chunkSize > len(dat)-offset {
			chunkSize = len(dat) - offset
			fin = true
		}
		chunk := dat[offset : offset+chunkSize]

		offset += chunkSize

		f := &WSFrame{
			Fin:  fin,
			Op:   op,
			Data: chunk,
		}
		frames = append(frames, f)
	}

	return conn.WriteFrames(frames)
}

type WSConn interface {
	ReadFrame() (*WSFrame, error)
	WriteFrame(*WSFrame) (int, error)
	WriteFrames([]*WSFrame) (int, error)
	io.Closer
	RemoteAddresser
}

type wsConn struct {
	conn net.Conn
	tx   *bufio.ReadWriter
	lckR *sync.Mutex
	lckW *sync.Mutex

	mask bool // true for websocket client, false for server.
}

var _ WSConn = (*wsConn)(nil)

func (t *wsConn) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (f *wsConn) ReadFrame() (*WSFrame, error) {
	f.lckR.Lock()
	defer f.lckR.Unlock()

	header := make([]byte, 2)
	if _, err := io.ReadFull(f.tx, header); err != nil {
		return nil, err
	}

	fin := header[0]&(1<<7) > 0
	opCode := header[0] & 0b1111
	if opCode == 8 {
		return &WSFrame{Fin: fin, Op: opCode}, io.EOF
	}

	masking := header[1]&(1<<7) > 0
	sizeBase := header[1] & 0b1111111
	size := 0
	if sizeBase < 126 {
		size = int(sizeBase)
	} else if sizeBase == 126 {
		sizeb := make([]byte, 2)
		if _, err := io.ReadFull(f.tx, sizeb); err != nil {
			return nil, err
		}
		size = int(binary.BigEndian.Uint16(sizeb))
	} else {
		sizeb := make([]byte, 8)
		if _, err := io.ReadFull(f.tx, sizeb); err != nil {
			return nil, err
		}
		size = int(binary.BigEndian.Uint64(sizeb))
	}

	if masking == f.mask {
		return nil, fmt.Errorf("mask error in receiving frame.")
	}
	mask := ([]byte)(nil)
	if masking {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(f.tx, mask); err != nil {
			return nil, err
		}
	}

	chunk := make([]byte, size)
	if _, err := io.ReadFull(f.tx, chunk); err != nil {
		return nil, err
	}

	if len(mask) > 0 {
		for i, v := range chunk {
			chunk[i] = v ^ mask[i%4]
		}
	}

	// slog.Debug(
	// 	"received frame:",
	// 	"fin", fin,
	// 	"op", opCode,
	// 	"len", len(chunk),
	// )

	return &WSFrame{Fin: fin, Op: opCode, Data: chunk}, nil
}

func (f *wsConn) WriteFrames(ml []*WSFrame) (int, error) {
	f.lckW.Lock()
	defer f.lckW.Unlock()

	sizeWrote := 0

	for _, m := range ml {
		if size, err := f.writeFrame(m); err != nil {
			return sizeWrote, err
		} else {
			sizeWrote += size
		}
	}

	return sizeWrote, nil
}

func (f *wsConn) WriteFrame(m *WSFrame) (int, error) {
	f.lckW.Lock()
	defer f.lckW.Unlock()

	return f.writeFrame(m)
}

func (f *wsConn) writeFrame(m *WSFrame) (int, error) {
	chunk := m.Data
	size := len(chunk)

	sizeExtra := 0
	if size > 125 {
		if size >= 1<<16 {
			sizeExtra = 8
		} else {
			sizeExtra = 2
		}
	}

	maskSize := 0
	if f.mask {
		maskSize = 4
	}

	header := make([]byte, 2+sizeExtra+maskSize)

	/* [FIN _ _ _ OP] */
	header[0] = 0b1111 & m.Op
	if m.Fin {
		header[0] = header[0] | 0b10000000
	}

	if size <= 125 {
		header[1] = byte(size)
	} else if sizeExtra == 2 {
		header[1] = byte(126)
		binary.BigEndian.PutUint16(header[2:4], uint16(size))
	} else if sizeExtra == 8 {
		header[1] = byte(127)
		binary.BigEndian.PutUint64(header[2:10], uint64(size))
	}
	if f.mask {
		header[1] = header[1] | (1 << 7)
	}

	if f.mask {
		mask := make([]byte, 4)
		rand.Read(mask)
		copy(header[len(header)-maskSize:], mask)
		for i, v := range chunk {
			chunk[i] = v ^ mask[i%4]
		}
	}

	chunkFin := make([]byte, len(header)+size)
	copy(chunkFin, header)
	copy(chunkFin[len(header):], chunk)

	if size, err := f.tx.Write(chunkFin); err == nil {
		f.tx.Flush()
		return size, nil
	} else {
		return size, err
	}
}

func (f *wsConn) Close() error {
	if f.conn != nil {
		f.conn.Close()
	}
	return nil
}
