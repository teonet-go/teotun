// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teonet tunnel is client/server application to creating regular tunnel between
// hosts without IPs.
package main

import (
	"flag"
	"time"

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
	// Application parameters
	var name = flag.String("name", appShort, "interface name")
	var port = flag.Int("p", 0, "local port number")
	var connectto = flag.String("connectto", "", "peer address to connect")
	var postcon = flag.String("postcon", "", "post connection shell command")
	var loglevel = flag.String("loglevel", "none", "log level")
	flag.Parse()

	// Application logo
	teonet.Logo(appName, appVersion)

	// Create new Teonet client
	teo, err := teonet.New(*name, *port, *loglevel)
	if err != nil {
		panic("can't create Teonet client")
	}

	// Connect to teonet
	for teo.Connect() != nil {
		teo.Log().Debug.Println("can't connect to Teonet, try again...")
		time.Sleep(1 * time.Second)
	}

	// Create new Teoonet tunnel
	_, err = teotun.New(teo, *name, *connectto, *postcon)
	if err != nil {
		panic("can't create Teoonet tunnel, " + err.Error())
	}

	select {}
}
