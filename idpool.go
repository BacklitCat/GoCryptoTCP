package GoCryptoTCP

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

type IDPool struct {
	mux       sync.Mutex
	pool      []int
	usingMap  map[int]bool
	cap       int
	available int
	assignI   int
	using     int
	recycled  int
}

func NewIDPool(n int) *IDPool {
	p := &IDPool{
		cap:       n,
		mux:       sync.Mutex{},
		pool:      make([]int, n),
		usingMap:  make(map[int]bool, n),
		using:     0,
		available: n,
		assignI:   0,
		recycled:  0,
	}
	p.initPool()
	return p
}

func (p *IDPool) initPool() {
	for i := 0; i < p.cap; i++ {
		p.pool[i] = i
	}
	p.shufflePool(p.cap)
}

func (p *IDPool) shufflePool(length int) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(length, func(i, j int) {
		p.pool[i], p.pool[j] = p.pool[j], p.pool[i]
	})
}

func (p *IDPool) Assign() (int, error) {
	if p.available == 0 {
		if p.recycled == 0 {
			return -1, errors.New("no available id to assign")
		}
		p.collate()
	}
	p.mux.Lock()
	newID := p.pool[p.assignI]
	p.usingMap[newID] = true
	p.available--
	p.assignI++
	p.using++
	p.mux.Unlock()
	return newID, nil
}

func (p *IDPool) collate() {
	p.mux.Lock()
	for i, j := 0, 0; i < p.cap; i++ {
		b, ok := p.usingMap[i]
		if !ok || !b {
			p.pool[j] = i
			j++
		}
	}
	p.available, p.assignI, p.recycled = p.available+p.recycled, 0, 0
	p.shufflePool(p.available)
	p.mux.Unlock()
}

func (p *IDPool) Recycle(oldId int) {
	p.mux.Lock()
	p.usingMap[oldId] = false
	p.recycled++
	p.using--
	p.mux.Unlock()
}
