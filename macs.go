// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teotun is package. Mac address manage module.

package teotun

import "sync"

// macaddr is tunnels connected mac address structure and methods receiver
type macaddr struct {
	*sync.RWMutex
	m map[string]string
}

// newMacaddr crete new macaddr object
func (t *Teotun) newMacaddr() (p *macaddr) {
	p = new(macaddr)
	p.RWMutex = new(sync.RWMutex)
	p.m = make(map[string]string)
	return
}

// add mac address
func (p *macaddr) add(mac, address string) {
	p.Lock()
	defer p.Unlock()
	p.m[mac] = address
}

// del mac address
func (p *macaddr) del(mac string) {
	p.Lock()
	defer p.Unlock()
	delete(p.m, mac)
}

// delByAddr delete mac by peer address
func (p *macaddr) delByAddr(address string) {
	p.Lock()
	defer p.Unlock()
	for mac, addr := range p.m {
		if addr == address {
			delete(p.m, mac)
		}
	}
}

// get mac address
func (p *macaddr) get(mac string) (address string, ok bool) {
	p.RLock()
	defer p.RUnlock()
	address, ok = p.m[mac]
	return
}

// forEach calls function f for each added mac address
func (p *macaddr) forEach(f func(mac, address string)) {
	p.RLock()
	defer p.RUnlock()
	for mac, address := range p.m {
		f(mac, address)
	}
}


