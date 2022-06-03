// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teotun package. Direct connect map module.

package teotun

import (
	"strings"
	"sync"
)

// directConnect is tunnels direct connect request structure and methods receiver
type directConnect struct {
	*sync.RWMutex
	m map[string]interface{}
}

// newDirectConnect crete new directConnect object
func (t *Teotun) newDirectConnect() (d *directConnect) {
	d = new(directConnect)
	d.RWMutex = new(sync.RWMutex)
	d.m = make(map[string]interface{})
	return
}

// add direct connect macs pair
func (d *directConnect) add(mac1, mac2 string) {
	d.Lock()
	defer d.Unlock()
	d.m[mac1+","+mac2] = nil
}

// del direct connect macs by one mac
func (d *directConnect) del(mac string) {
	d.Lock()
	defer d.Unlock()

	for key := range d.m {
		macs := strings.Split(key, ",")
		if len(macs) < 2 {
			return
		}
		if macs[0] == mac || macs[1] == mac {
			delete(d.m, key)
		}
	}
}

// get peer
func (d *directConnect) get(mac1, mac2 string) (v interface{}, ok bool) {
	d.RLock()
	defer d.RUnlock()

	keys := []string{mac1 + "," + mac2, mac2 + "," + mac1}
	for _, key := range keys {
		v, ok = d.m[key]
		if ok {
			return
		}
	}

	return
}
