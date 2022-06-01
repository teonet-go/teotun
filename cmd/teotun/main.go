// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet tunnel is client/server application to creating regular tunnel between
// hosts without IPs.
package main

import (
	"github.com/teonet-go/teonet"
	"github.com/teonet-go/teotun"
)

const (
	appName    = "Teonet tunnel"
	appShort   = "teotun"
	appLong    = ""
	appVersion = "0.0.1"
)

func main() {
	// Application logo
	teonet.Logo(appName, appVersion)

	// Create new Teonet client
	teo, err := teonet.New(appShort)
	if err != nil {
		panic("can't create Teonet client")
	}

	// Create new Teonun object
	teotun.New(teo)

	select {}
}
