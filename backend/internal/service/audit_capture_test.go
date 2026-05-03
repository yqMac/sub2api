package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditCaptureBuffer_WriteUnderLimit(t *testing.T) {
	b := AcquireAuditCaptureBuffer(100)
	defer ReleaseAuditCaptureBuffer(b)

	n, err := b.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, 5, b.Total)
	assert.Equal(t, "hello", string(b.Bytes()))
	assert.False(t, b.Truncated())
}

func TestAuditCaptureBuffer_WriteOverLimit(t *testing.T) {
	b := AcquireAuditCaptureBuffer(5)
	defer ReleaseAuditCaptureBuffer(b)

	b.Write([]byte("hello"))
	b.Write([]byte("world"))

	assert.Equal(t, 10, b.Total)
	assert.Equal(t, "hello", string(b.Bytes()))
	assert.True(t, b.Truncated())
}

func TestAuditCaptureBuffer_PartialWrite(t *testing.T) {
	b := AcquireAuditCaptureBuffer(3)
	defer ReleaseAuditCaptureBuffer(b)

	b.Write([]byte("ab"))
	b.Write([]byte("cdef"))

	assert.Equal(t, 6, b.Total)
	assert.Equal(t, "abc", string(b.Bytes()))
	assert.True(t, b.Truncated())
}

func TestAuditCaptureBuffer_EmptyWrite(t *testing.T) {
	b := AcquireAuditCaptureBuffer(10)
	defer ReleaseAuditCaptureBuffer(b)

	n, err := b.Write(nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Equal(t, 0, b.Total)
	assert.False(t, b.Truncated())
}

func TestAuditCaptureBuffer_ExactLimit(t *testing.T) {
	b := AcquireAuditCaptureBuffer(5)
	defer ReleaseAuditCaptureBuffer(b)

	b.Write([]byte("12345"))
	assert.Equal(t, 5, b.Total)
	assert.Equal(t, 5, b.Len())
	assert.False(t, b.Truncated())

	b.Write([]byte("6"))
	assert.Equal(t, 6, b.Total)
	assert.Equal(t, 5, b.Len())
	assert.True(t, b.Truncated())
}

func TestReleaseAuditCaptureBuffer_Nil(t *testing.T) {
	ReleaseAuditCaptureBuffer(nil)
}
