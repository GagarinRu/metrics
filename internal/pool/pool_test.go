package pool_test

import (
	"testing"

	"github.com/GagarinRu/metrics/internal/pool"
	"github.com/stretchr/testify/require"
)

type item struct {
	value int
}

func (i *item) Reset() {
	i.value = 0
}

func TestPoolGetPut(t *testing.T) {
	p := pool.New[*item]()

	obj := &item{value: 42}
	p.Put(obj)

	got := p.Get()
	require.Equal(t, 0, got.value)

	got.value = 100
	p.Put(got)

	got2 := p.Get()
	require.Equal(t, 0, got2.value)
}

func TestPoolGetEmpty(t *testing.T) {
	p := pool.New[*item]()
	got := p.Get()
	require.NotNil(t, got)
	require.Equal(t, 0, got.value)
}
