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
	appVersion = teotun.Version
)

func main() {
	// Application parameters
	var name = flag.String("name", appShort, "interface name")
	var port = flag.Int("p", 0, "local port number")
	var connectto = flag.String("connectto", "", "peer address to connect")
	var postcon = flag.String("postcon", "", "post connection shell command")
	var loglevel = flag.String("loglevel", "none", "log level")
	var logfilter = flag.String("logfilter", "", "set log filter")
	var hotkey = flag.Bool("hotkey", false, "start hotkey menu")
	var stat = flag.Bool("stat", false, "print statistic")

	flag.Parse()

	// Application logo
	teonet.Logo(appName, appVersion)

	// Create new Teonet client
	teo, err := teonet.New(*name, *port, *loglevel, teonet.Logfilter(*logfilter),
		teonet.Hotkey(*hotkey), teonet.Stat(*stat))
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
