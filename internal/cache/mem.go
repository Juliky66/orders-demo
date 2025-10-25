package cache

import (
	"sync"

	"order/internal/model"
)

type Mem struct{ m sync.Map }

func NewMem() *Mem { return &Mem{} }

func (c *Mem) Set(id string, o model.Order) { c.m.Store(id, o) }

func (c *Mem) Get(id string) (model.Order, bool) {
	v, ok := c.m.Load(id)
	if !ok {
		return model.Order{}, false
	}
	return v.(model.Order), true
}

func (c *Mem) Range(fn func(string, model.Order) bool) {
	c.m.Range(func(k, v any) bool { return fn(k.(string), v.(model.Order)) })
}
