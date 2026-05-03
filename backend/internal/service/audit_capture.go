// [bmai-fork] capped buffer for capturing streaming response previews
package service

import (
	"bytes"
	"sync"
)

// AuditCaptureBuffer captures up to Limit bytes. Further writes are dropped
// but Total still increments so callers know the real size.
type AuditCaptureBuffer struct {
	buf   *bytes.Buffer
	Total int
	Limit int
}

var auditCapturePool = sync.Pool{
	New: func() any {
		return &AuditCaptureBuffer{buf: bytes.NewBuffer(make([]byte, 0, 4096))}
	},
}

func AcquireAuditCaptureBuffer(limit int) *AuditCaptureBuffer {
	b := auditCapturePool.Get().(*AuditCaptureBuffer)
	b.buf.Reset()
	b.Total = 0
	b.Limit = limit
	return b
}

func ReleaseAuditCaptureBuffer(b *AuditCaptureBuffer) {
	if b == nil {
		return
	}
	if b.buf.Cap() > 256*1024 {
		b.buf = bytes.NewBuffer(make([]byte, 0, 4096))
	}
	auditCapturePool.Put(b)
}

func (b *AuditCaptureBuffer) Write(p []byte) (int, error) {
	n := len(p)
	b.Total += n
	if b.buf.Len() >= b.Limit {
		return n, nil
	}
	remain := b.Limit - b.buf.Len()
	if remain >= n {
		b.buf.Write(p)
	} else {
		b.buf.Write(p[:remain])
	}
	return n, nil
}

func (b *AuditCaptureBuffer) Bytes() []byte   { return b.buf.Bytes() }
func (b *AuditCaptureBuffer) Len() int         { return b.buf.Len() }
func (b *AuditCaptureBuffer) Truncated() bool  { return b.Total > b.buf.Len() }
