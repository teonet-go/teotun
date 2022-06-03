// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teotun package. Peers manage module.

package teotun

import "sync"

// peers is tunnels connected peers structure and methods receiver
type peers struct {
	*sync.RWMutex
	m map[string]interface{}
}

// newPeers crete new peers object
func (t *Teotun) newPeers() (p *peers) {
	p = new(peers)
	p.RWMutex = new(sync.RWMutex)
	p.m = make(map[string]interface{})
	return
}

// add peer
func (p *peers) add(address string) {
	p.Lock()
	defer p.Unlock()
	p.m[address] = nil
}

// del peer
func (p *peers) del(address string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, address)
}

// get peer
func (p *peers) get(address string) (v interface{}, ok bool) {
	p.RLock()
	defer p.RUnlock()
	v, ok = p.m[address]
	return
}

// forEach calls function f for each added peer
func (p *peers) forEach(f func(address string)) {
	p.RLock()
	defer p.RUnlock()
	for address := range p.m {
		f(address)
	}
}
