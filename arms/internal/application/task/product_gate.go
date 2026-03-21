package task

import (
	"sync"

	"github.com/closeloopautomous/arms/internal/domain"
)

// ProductGate serializes completion (and similar) work per product on this process.
type ProductGate struct {
	mu sync.Mutex
	by map[domain.ProductID]*sync.Mutex
}

func NewProductGate() *ProductGate {
	return &ProductGate{by: make(map[domain.ProductID]*sync.Mutex)}
}

func (g *ProductGate) WithLock(productID domain.ProductID, fn func() error) error {
	m := g.mutexFor(productID)
	m.Lock()
	defer m.Unlock()
	return fn()
}

func (g *ProductGate) mutexFor(productID domain.ProductID) *sync.Mutex {
	g.mu.Lock()
	defer g.mu.Unlock()
	m, ok := g.by[productID]
	if !ok {
		m = &sync.Mutex{}
		g.by[productID] = m
	}
	return m
}
