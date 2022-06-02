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

const Version = "0.0.3"

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
	macs  *macaddr
}

// Create new Teonet tunnel, where:
//  teo - connected teonet client
//  iface - tunnels interface name
//  connectto - remote peer teonet address to connet
//  postcon - post connection shell command
func New(teo *teonet.Teonet, iface, connectto, postcon string) (t *Teotun, err error) {

	// Create
	t = new(Teotun)
	t.teo = teo
	t.log = teo.Log()
	t.peers = t.newPeers()
	t.macs = t.newMacaddr()

	// Create tap interface
	err = t.ifcCreate(iface)
	if err != nil {
		err = errors.New("can't create interface, error: " + err.Error())
		return
	}

	// Connect to remote peer if connectto parameter sets. And process Teonet
	// connection
	err = t.teoConnect(connectto)
	if err != nil {
		err = errors.New("can't connect Teonet, error: " + err.Error())
		return
	}

	// Exec post connection commands
	err = t.postConnect(postcon)

	return
}

// ifcCreate create tap interface and start procces it
func (t *Teotun) ifcCreate(name string) (err error) {

	// Configure interface
	config := water.Config{
		DeviceType: water.TAP,
	}
	config.Name = name

	// Create interface
	ifce, err := water.New(config)
	if err != nil {
		return
	}
	t.ifce = ifce

	// Read from interface and send to teonet peers
	go t.ifcProcess()

	return
}

// ifcProcess get frame from interface and send it to teonet peers
func (t *Teotun) ifcProcess() {
	var frame ethernet.Frame
	for {
		frame.Resize(1500)
		n, err := t.ifce.Read([]byte(frame))
		if err != nil {
			t.log.Error.Fatal(err)
		}
		frame = frame[:n]
		dest := frame.Destination().String()
		t.log.Debug.Println("Got from interface:")
		t.log.Debug.Printf("Dst: %s\n", dest)
		t.log.Debug.Printf("Src: %s\n", frame.Source())
		t.log.Debug.Printf("Ethertype: % x\n", frame.Ethertype())
		t.log.Debug.Printf("Payload len: %d\n", len(frame.Payload()))

		// Resend frame to tunnels peer by teonet address found by mac address
		if address, ok := t.macs.get(dest); ok {
			t.teo.Command(cmdData, []byte(frame)).SendTo(address)
			continue
		}

		// Resend frame to all connected tunnels peers
		t.peers.forEach(func(address string) {
			t.teo.Command(cmdData, []byte(frame)).SendTo(address)
		})
	}
}

// teoConnect function connect to peer by address if connectto sets.
// And start listen commands 11 and 12:
//   the cmd=11 is connectto command
//   the cmd=12 is remote tunnel data
func (t *Teotun) teoConnect(address string) (err error) {

	var clientMode = len(address) > 0

	// Set reader to process teonet events and commands
	t.teo.AddReader(func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {
		return t.teoProcess(clientMode, address, c, p, e)
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

// teoProcess get message from peer, resend it to other peers or send it to
// interface
func (t *Teotun) teoProcess(clientMode bool, address string,
	c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {

	var addr = c.Address()

	// Check received teonet event
	switch {

	// Check Peer Disconnected event and remove it from peers in client mode
	case e.Event == teonet.EventDisconnected:
		if _, ok := t.peers.get(addr); !ok {
			return false
		}

		t.log.Connect.Printf(
			"peer %s disconnected from tunnel (event disconnected)",
			addr,
		)

		// Remmove peer from peers and masc lists
		t.peers.del(addr)
		t.macs.deladdr(addr)

		return false

	// Check Peer Connected event
	case e.Event == teonet.EventConnected:
		t.log.Connect.Printf("peer %s connected to tunnel (event connected)",
			addr)
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
		t.peers.add(addr)
		t.log.Connect.Printf("got cmd connect from peer %s\n", addr)
		// Send answer in server mode
		if !clientMode {
			cmd.Send(c)
		}

	// Get data command
	case cmdData:
		// Check connected
		_, ok := t.peers.get(addr)
		if !ok {
			t.log.Error.Printf("receve data packet from unknown peer %s\n",
				addr)
			t.teo.CloseTo(addr)
			break
		}

		// Show log
		var frame ethernet.Frame = cmd.Data
		var src = frame.Source().String()
		var dst = frame.Destination().String()
		t.log.Debug.Printf("Got from teonet address %s:\n", addr)
		t.log.Debug.Printf("Dst: %s\n", dst)
		t.log.Debug.Printf("Src: %s\n", src)
		t.log.Debug.Printf("Ethertype: % x\n", frame.Ethertype())
		t.log.Debug.Printf("Payload len: %d\n", len(frame.Payload()))

		// Save source mac address
		t.macs.add(src, addr)

		// Check destination
		switch {
		// Brodcast request
		case dst == "ff:ff:ff:ff:ff:ff":
			// send to all connected peers except this
			t.peers.forEach(func(address string) {
				if address != addr {
					t.teo.Command(cmdData, []byte(frame)).SendTo(address)
				}
			})
			// does not return or break here, because we need to send this
			// frame to the interface too

		// Check if request send to other peer then resend frame to peer
		default:
			if address, ok := t.macs.get(dst); ok {
				t.teo.Command(cmdData, []byte(frame)).SendTo(address)
				return true
			}
		}

		// Send data to tunnel interface
		// TODO: wait ifce ready
		// for t.ifce == nil {
		// 	time.Sleep(10 * time.Millisecond)
		// }
		t.ifce.Write([]byte(frame))
	}

	return true
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
