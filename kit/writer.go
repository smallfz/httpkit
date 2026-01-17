package kit

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
)

type monitoredWriter struct {
	base        http.ResponseWriter
	headerWrote *atomic.Bool
	hijacked    *atomic.Bool
}

var _ http.ResponseWriter = (*monitoredWriter)(nil)
var _ http.Flusher = (*monitoredWriter)(nil)
var _ http.Hijacker = (*monitoredWriter)(nil)

func newMonitoredWriter(base http.ResponseWriter) *monitoredWriter {
	return &monitoredWriter{
		base:        base,
		headerWrote: new(atomic.Bool),
		hijacked:    new(atomic.Bool),
	}
}

func (w *monitoredWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.base.(http.Hijacker); ok {
		if w.hijacked.CompareAndSwap(false, true) {
			if conn, rw, err := hj.Hijack(); err == nil {
				return conn, rw, nil
			} else {
				return nil, nil, err
			}
		}
		return nil, nil, fmt.Errorf("already hijacked.")
	}
	return nil, nil, fmt.Errorf("%T doesn't support hijacking.", w.base)
}

func (w *monitoredWriter) Flush() {
	if flusher, ok := w.base.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *monitoredWriter) Header() http.Header {
	return w.base.Header()
}

func (w *monitoredWriter) Write(dat []byte) (int, error) {
	w.headerWrote.CompareAndSwap(false, true)
	return w.base.Write(dat)
}

func (w *monitoredWriter) WriteHeader(code int) {
	if w.hijacked.Load() {
		return
	}
	if w.headerWrote.CompareAndSwap(false, true) {
		w.base.WriteHeader(code)
	}
}
