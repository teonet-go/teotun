// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Teotun is package for creating client/server application which make regular
// tunnel between hosts without IPs based on Teonet
package teotun

import (
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
	"github.com/teonet-go/teonet"
	"github.com/teonet-go/tru/teolog"
)

const Version = "0.0.2"

const (
	cmdConnect = 11 // connect and connect_answer command
	cmdData    = 12 // data command
)

// Teotun is main package methods holder and data structure type
type Teotun struct {
	teo   *teonet.Teonet
	log   *teolog.Teolog
	ifce  *water.Interface
	peers *peers
}

// Create new Teonet tunnel, where:
//  teo - connected teonet client
//  iface - tunnels interface name
//  connectto - remote peer teonet address to connet
//  postcon - post connection shell command
func New(teo *teonet.Teonet, iface, connectto string, postcon string) (t *Teotun, err error) {

	// Create
	t = new(Teotun)
	t.teo = teo
	t.log = teo.Log()
	t.peers = t.newPeers()

	// Create tap interface
	t.ifce, err = t.ifcCreate(iface)
	if err != nil {
		err = errors.New("can't create interface, error: " + err.Error())
		return
	}

	// Connect to remote peer if connectto parameter sets. And process Teonet
	// connection
	err = t.teoConnect(connectto)
	if err != nil {
		return
	}

	// Exec post connection commands
	err = t.postConnect(postcon)

	return
}

// ifcCreate create tap interface
func (t *Teotun) ifcCreate(name string) (ifce *water.Interface, err error) {
	config := water.Config{
		DeviceType: water.TAP,
	}
	config.Name = name

	// Create interface
	ifce, err = water.New(config)
	if err != nil {
		return
	}

	// Read from interface and send to tru channels
	go func() {
		var frame ethernet.Frame
		for {
			frame.Resize(1500)
			n, err := ifce.Read([]byte(frame))
			if err != nil {
				t.log.Error.Fatal(err)
			}
			frame = frame[:n]
			t.log.Debug.Println("Got from interface:")
			t.log.Debug.Printf("Dst: %s\n", frame.Destination())
			t.log.Debug.Printf("Src: %s\n", frame.Source())
			t.log.Debug.Printf("Ethertype: % x\n", frame.Ethertype())
			t.log.Debug.Printf("Payload len: %d\n", len(frame.Payload()))

			// TODO: Now it resend frames to all connected tunnels peers. But it
			// should send to peers depend of frame.Destination() field
			t.peers.forEach(func(address string) {
				t.teo.Command(cmdData, []byte(frame)).SendTo(address)
			})
		}
	}()

	return
}

// teoConnect function connect to peer by address if connectto sets.
// And start listen commands 11 and 12:
//   the cmd=11 is connectto command
//   the cmd=12 is remote tunnel data
func (t *Teotun) teoConnect(address string) (err error) {

	var clientMode = len(address) > 0

	// Set reader to process teonet events and commands
	t.teo.AddReader(func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {

		// Check received events
		switch {

		// Check peer disconnected event and remove it from peers in client mode
		case e.Event == teonet.EventDisconnected:
			t.log.Connect.Printf("peer %s disconnected from tunnel (event disconnected)",
				c.Address())
			if clientMode && c.Address() == address {
				t.peers.del(address)
			}
			return false

		// Check peer connected event
		case e.Event == teonet.EventConnected:
			t.log.Connect.Printf("peer %s connected to tunnel (event connected)",
				c.Address())
			// Send connect command
			if clientMode {
				t.teo.Command(cmdConnect, nil).SendTo(address)
			}
			return false

		// Skip non data events
		case e.Event != teonet.EventData:
			return false
		}

		// Parse incomming commands
		cmd := t.teo.Command(p.Data())
		switch cmd.Cmd {

		// Connect command
		case cmdConnect:
			// Add peers address to connected peers
			t.peers.add(c.Address())
			t.log.Connect.Printf("got cmd connect from peer %s\n", c.Address())
			// Send answer in server mode
			if !clientMode {
				cmd.Send(c)
			}

		// Get data command
		case cmdData:
			// Check connected
			_, ok := t.peers.get(c.Address())
			if !ok {
				t.log.Error.Printf("receve data packet from unknown peer %s\n",
					c.Address())
				break
			}

			// Show log
			var frame ethernet.Frame = cmd.Data
			t.log.Debug.Printf("Got from teonet address %s:\n", c.Address())
			t.log.Debug.Printf("Dst: %s\n", frame.Destination())
			t.log.Debug.Printf("Src: %s\n", frame.Source())
			t.log.Debug.Printf("Ethertype: % x\n", frame.Ethertype())
			t.log.Debug.Printf("Payload len: %d\n", len(frame.Payload()))

			// Send data to tunnel interface
			// TODO: wait ifce ready
			for t.ifce == nil {
				time.Sleep(10 * time.Millisecond)
			}
			t.ifce.Write(cmd.Data)
		}

		return false
	})

	// Connect to remote peer and send connect command
	if clientMode {
		// Connect to remote peer
		for t.teo.ConnectTo(address) != nil {
			t.log.Error.Printf("can't connect to %s, try again...", address)
			time.Sleep(1 * time.Second)
		}
	}

	return
}

// postConnect execute post connection shell command
func (t *Teotun) postConnect(command string) (err error) {
	if len(command) == 0 {
		return
	}
	com := strings.Split(command, " ")
	var arg []string
	if len(com) > 1 {
		arg = com[1:]
	}

	out, err := exec.Command(com[0], arg...).Output()
	t.log.Debug.Printf("\n%s\n", out)
	return
}
