// Package pool provides a generic object pool with Reset support.
package pool

import (
	"reflect"
	"sync"
)

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	pool sync.Pool
}

func New[T Resetter]() *Pool[T] {
	p := &Pool[T]{}
	p.pool.New = func() any {
		return newValue[T]()
	}
	return p
}

func newValue[T Resetter]() T {
	var v T
	rv := reflect.ValueOf(&v).Elem()
	if rv.Kind() == reflect.Pointer && rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}
	return v
}

func (p *Pool[T]) Get() T {
	v := p.pool.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

func (p *Pool[T]) Put(x T) {
	x.Reset()
	p.pool.Put(x)
}
