// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teotun is package for creating client/server application which make regular
// tunnel between hosts without IPs based on Teonet
package teotun

import "github.com/teonet-go/teonet"

// Teonun is main package methods holder and data structure type
type Teonun struct {
}

// Create new Teonun object
func New(teo *teonet.Teonet) *Teonun {
	return new(Teonun)
}
