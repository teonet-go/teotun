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

const Version = "0.0.4"

const (
	cmdConnect       = 11 // connect and connect_answer command
	cmdData          = 12 // data command
	cmdDirectConnect = 13 // direct connect command: <addr>,<mac>
)

// Teotun is main package methods holder and data structure type
type Teotun struct {
	teo   *teonet.Teonet
	log   *teolog.Teolog
	ifce  *water.Interface
	peers *peers
	macs  *macaddr
	dcmap *directConnect
	dc    *sync.Mutex
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
	t.dcmap = t.newDirectConnect()
	t.dc = new(sync.Mutex)

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
		src := frame.Source().String()
		dst := frame.Destination().String()
		t.log.Debug.Println("Got from interface:")
		t.log.Debug.Printf("Dst: %s\n", dst)
		t.log.Debug.Printf("Src: %s\n", src)
		t.log.Debug.Printf("Ethertype: % x\n", frame.Ethertype())
		t.log.Debug.Printf("Payload len: %d\n", len(frame.Payload()))

		// Resend frame to tunnels peer by teonet address found by mac address
		if address, ok := t.macs.get(dst); ok {
			saddr, _ := t.macs.get(src)
			t.log.Debug.Printf(
				"Resend (interface) from src: %s to dst: %s\n%s -> %s",
				src, dst, saddr, address,
			)
			t.teo.Command(cmdData, []byte(frame)).SendTo(address)
			continue
		}

		// Resend frame to all connected tunnels peers
		t.peers.forEach(func(address string) {
			t.log.Debug.Printf("Resend (interface) to all, %s\n", address)
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

	// Process teonet event
	if !t.teoEvents(e.Event, addr, clientMode && addr == address) {
		return false
	}

	// Process teonet commands
	return t.teoCommands(p, c, addr, clientMode)
}

// teoEvents proccess Teonet Events, return true if it data event
func (t *Teotun) teoEvents(event teonet.TeonetEventType, addr string, clientMode bool) bool {
	switch event {

	// Connected event
	case teonet.EventConnected:
		t.teoEventConnect(addr, clientMode)

	// Disconnected event
	case teonet.EventDisconnected:
		t.teoEventDiconnect(addr)

	// Data events
	case event:
		return true
	}
	return false
}

// teoEventConnect process Teonet Connected Event
func (t *Teotun) teoEventConnect(addr string, clientMode bool) {
	t.log.Connect.Printf(
		"peer %s connected to tunnel (event connected)", addr,
	)
	// Send connect command to trutun server selected in app parameters when
	// connected or reconnected to this server
	if clientMode {
		t.teo.Command(cmdConnect, nil).SendTo(addr)
	}
}

// teoEventDiconnect process Teonet Diconnect Event
func (t *Teotun) teoEventDiconnect(addr string) {
	if _, ok := t.peers.get(addr); !ok {
		return
	}

	t.log.Connect.Printf(
		"peer %s disconnected from tunnel (event disconnected)",
		addr,
	)

	// Remmove peer from peers and masc lists
	t.peers.del(addr)
	t.macs.forEach(func(mac, addr string) {
		t.dcmap.del(mac)
	})
	t.macs.delByAddr(addr)
}

// teoCommands proccess Teonet Commands
func (t *Teotun) teoCommands(p *teonet.Packet, c *teonet.Channel, addr string, clientMode bool) bool {
	cmd := t.teo.Command(p.Data())
	switch cmd.Cmd {

	// Connect command
	case cmdConnect:
		t.teoCommandConnect(cmd, c)

	// Direct connect to peer command
	case cmdDirectConnect:
		t.teoCommandDirectConnect(cmd, addr)

	// Get data command
	case cmdData:
		return t.teoCommandGetData(cmd, addr)

	// Skip any other commands
	default:
		return false
	}
	return true
}

// teoCommandConnect process Connect Command
func (t *Teotun) teoCommandConnect(cmd *teonet.Command, c *teonet.Channel) {
	var addr = c.Address()
	t.log.Connect.Printf("got cmd connect from peer %s\n", addr)
	if _, ok := t.peers.get(addr); ok {
		return
	}
	// Add peers address to connected peers
	t.peers.add(addr)
	cmd.Send(c)
}

// teoCommandDirectConnect process Direct Connect Command
func (t *Teotun) teoCommandDirectConnect(cmd *teonet.Command, addr string) (err error) {

	// Parse data
	params := strings.Split(string(cmd.Data), ",")
	if len(params) < 4 {
		err = errors.New("worng data")
		return
	}
	dstaddr := params[0]
	dstmac := params[1]
	srcaddr := params[2]
	srcmac := params[3]
	t.log.Connect.Printf(
		"got cmd direct connect from peer %s to %s\n", addr, dstaddr,
	)

	// Skip if already connected
	if t.teo.Connected(dstaddr) {
		t.log.Connect.Printf("skip direct connect, already connected\n")
		// Set mac address
		// t.teo.Command(cmdConnect, nil).SendTo(dstaddr)
		t.macs.add(dstmac, dstaddr)
		err = errors.New("already connected")
		return
	}

	// Connect to peer
	go func() {
		if !t.dc.TryLock() {
			return
		}
		defer t.dc.Unlock()

		// Send connect command to dst peer
		if err := t.teo.ConnectTo(dstaddr); err != nil {
			return
		}
		t.teo.ReconnectOff(dstaddr)
		t.teo.Command(cmdConnect, nil).SendTo(dstaddr)
		t.macs.add(dstmac, dstaddr)

		// Send DirectConnect to dst peer
		data := srcaddr + "," + srcmac + "," + dstaddr + "," + dstmac
		t.teo.Command(cmdDirectConnect, data).SendTo(dstaddr)
		t.teo.Log().Debug.Printf(
			"send direct connect to %s, data: %s\n", dstaddr, data,
		)
	}()
	return
}

// teoCommandGetData process Get Data Command
func (t *Teotun) teoCommandGetData(cmd *teonet.Command, addr string) bool {
	// Check connected
	_, ok := t.peers.get(addr)
	if !ok {
		t.log.Error.Printf("receve data packet from unknown peer %s\n",
			addr)
		// t.teo.CloseTo(addr)
		return false
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

		// Exit from this switch if dst mac not found in macs map
		dstaddr, ok := t.macs.get(dst)
		if !ok {
			break
		}

		// Send frame to peer
		t.teo.Command(cmdData, []byte(frame)).SendTo(dstaddr)

		// Skip if direct connect already sent
		if _, ok := t.dcmap.get(src, dst); ok {
			return true
		}

		// Send direct connect command to peer
		if srcaddr, ok := t.macs.get(src); ok {
			t.dcmap.add(src, dst)

			// Send DirectConnect to src peer
			data := dstaddr + "," + dst + "," + srcaddr + "," + src
			t.teo.Command(cmdDirectConnect, data).SendTo(srcaddr)
			t.teo.Log().Debug.Printf(
				"send direct connect to %s, data: %s\n", src, data,
			)
		}
		return true
	}

	// Send data to tunnel interface
	// TODO: wait ifce ready
	// for t.ifce == nil {
	// 	time.Sleep(10 * time.Millisecond)
	// }
	t.ifce.Write([]byte(frame))

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
