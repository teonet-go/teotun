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
	"sync"
	"time"

	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
	"github.com/teonet-go/teonet"
	"github.com/teonet-go/tru/teolog"
)

const (
	cmdConnect = 11 // connect / connect_answer command
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
		// err = errors.New("can't create interface, error: " + err.Error())
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
			t.log.Debug.Printf("Dst: %s\n", frame.Destination())
			t.log.Debug.Printf("Src: %s\n", frame.Source())
			t.log.Debug.Printf("Ethertype: % x\n", frame.Ethertype())
			t.log.Debug.Printf("Payload: % x\n", frame.Payload())

			// TODO: Resend frame to all channels
			// for t.tru == nil {
			// 	time.Sleep(10 * time.Millisecond)
			// }
			t.peers.forEach(func(address string) {
				t.teo.Command(cmdData, frame).SendTo(address)
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

	// Connect to remote peer and send connect command
	if clientMode {
		// Connect to remote peer
		for t.teo.ConnectTo(address) != nil {
			t.log.Error.Printf("can't connect to %s, try again...", address)
			time.Sleep(1 * time.Second)
		}

		// Send connect command
		t.teo.Command(cmdConnect, nil).SendTo(address)
	}

	// Set reader to process teonet commands
	t.teo.AddReader(func(c *teonet.Channel, p *teonet.Packet, e *teonet.Event) bool {

		// Skip non data events
		if e.Event != teonet.EventData {
			return false
		}

		// Parse incomming commands
		cmd := t.teo.Command(p.Data())
		switch cmd.Cmd {

		// Connect command
		case cmdConnect:
			// Add peers address to connected peers
			t.peers.add(c.Address())
			t.log.Connect.Printf("peer %s connected to tunnel", c.Address())
			// Send answer in server mode
			if !clientMode {
				cmd.Send(c)
			}

		// Get data command
		case cmdData:
			// Show log
			t.log.Debug.Printf("got %d byte from %s, id %d: % x\n",
				p.Len(), c.Address(), p.ID(), cmd.Data)
			// Send data to tunnel interface
			// TODO: wait ifce ready
			for t.ifce == nil {
				time.Sleep(10 * time.Millisecond)
			}
			t.ifce.Write(cmd.Data)
		}

		return false
	})

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
func (p *peers) get(address string) (ok bool) {
	p.RLock()
	defer p.RUnlock()
	_, ok = p.m[address]
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
