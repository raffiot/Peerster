package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/dedis/protobuf"
)

/**
Constructor of Gossiper
*/
func NewGossiper(address string, name string, peers []string, simple bool, client string, timer int) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	udpConn, err := net.ListenUDP("udp4", udpAddr)

	if err != nil {
		fmt.Println("Error creating new gossiper !")
		log.Fatal(err)
	}
	udpAddrClient, err := net.ResolveUDPAddr("udp4", client)
	udpConnClient, err := net.ListenUDP("udp4", udpAddrClient)

	if err != nil {
		fmt.Println("Error creating new gossiper !")
		log.Fatal(err)
	}

	var s []PeerStatus
	var a []PeerMessage
	var pa = make(map[string][]PrivateMessage)

	elementMap := make(map[string]bool)
	for i := 0; i < len(peers); i++ {
		elementMap[peers[i]] = true
	}
	fmt.Println(elementMap)
	dsdv := make(map[string]string)
	return &Gossiper{
		udp_address:      udpAddr,
		conn:             udpConn,
		Name:             name,
		set_of_peers:     elementMap,
		vector_clock:     s,
		archives:         a,
		archives_private: pa,
		DSDV:             dsdv,
		simple:           simple,
		clientConn:       udpConnClient,
		clientAddr:       udpAddrClient,
		rtimer:           timer,
	}
}

/**
Routine that handle the messages that we receive from the client
at <127.0.0.1:ClientPort>
We handle only the GossipPacket of type SimpleMessage
*/
func (g *Gossiper) receiveMessageFromClient() {

	defer g.clientConn.Close()

	for {
		var pkt GossipPacket = GossipPacket{}
		b := make([]byte, UDP_PACKET_SIZE)
		nb_byte_written, _, err := g.clientConn.ReadFromUDP(b)
		protobuf.Decode(b, &pkt)
		if nb_byte_written > 0 && err == nil {
			fmt.Println(pkt.Simple != nil)
			fmt.Println(pkt.Private != nil)
			if pkt.Simple != nil {
				g.handleSimplePacket(pkt.Simple)
			} else if pkt.Private != nil {
				mutex.Lock()
				newPkt := GossipPacket{Private: &PrivateMessage{
					Origin:      g.Name,
					ID:          0,
					Text:        pkt.Private.Text,
					Destination: pkt.Private.Destination,
					HopLimit:    HOP_LIMIT,
				}}
				fmt.Println(pkt.Private.Destination)
				next_hop, ok := g.DSDV[pkt.Private.Destination]
				mutex.Unlock()
				if ok {
					_, ok := g.archives_private[pkt.Private.Destination]
					if !ok {
						var new_array []PrivateMessage
						g.archives_private[pkt.Private.Destination] = new_array
					}
					g.archives_private[pkt.Private.Destination] = append(g.archives_private[pkt.Private.Destination], *newPkt.Private)

					pktByte, err := protobuf.Encode(&newPkt)
					if err != nil {
						fmt.Println("Encode of the packet failed")
						log.Fatal(err)
					}
					mutex.Lock()
					g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
					mutex.Unlock()
					fmt.Println("private message send")
				} else {
					fmt.Println("destination unknown for private message")
				}
			} else {
				fmt.Println(pkt.Private)
				fmt.Println("Error client send packet with type different than simple message or private message.")
			}
		}
	}
}

/**
Function called when we receive a SimpleMessage from client
if we are in simple mode:
 	we send this SimpleMessage to all neighbors
if we aren't in simple mode:
	We take from the vector clock datastructure the ID of this new message
	We send this message as RumorMessage to the gossiper connexion
*/
func (g *Gossiper) handleSimplePacket(pkt *SimpleMessage) {
	fmt.Println("CLIENT MESSAGE " + pkt.Contents)
	if g.simple {
		newPkt := GossipPacket{Simple: &SimpleMessage{
			OriginalName:  g.Name,
			RelayPeerAddr: ParseIPStr(g.udp_address),
			Contents:      pkt.Contents,
		}}

		pktByte, err := protobuf.Encode(&newPkt)
		if err != nil {
			fmt.Println("Encode of the packet failed")
			log.Fatal(err)
		}
		g.listAllKnownPeers()
		mutex.Lock()
		for k := range g.set_of_peers {
			dst, err := net.ResolveUDPAddr("udp4", k)
			if err != nil {
				fmt.Println("cannot resolve addr of ather gossiper")
				log.Fatal(err)
			}
			g.conn.WriteToUDP(pktByte, dst)
		}
		mutex.Unlock()
	} else {
		var my_ID uint32 = 1
		mutex.Lock()
		for i := 0; i < len(g.vector_clock); i++ {
			if g.vector_clock[i].Identifier == g.Name {
				my_ID = g.vector_clock[i].NextID
			}
		}
		mutex.Unlock()
		newPkt := GossipPacket{Rumor: &RumorMessage{
			Origin: g.Name,
			ID:     my_ID,
			Text:   pkt.Contents,
		}}
		pktByte, err := protobuf.Encode(&newPkt)
		if err != nil {
			fmt.Println("Encode of the packet failed")
			log.Fatal(err)
		}
		mutex.Lock()
		g.conn.WriteToUDP(pktByte, g.udp_address)
		mutex.Unlock()
	}
	return
}

/**
This function handle SimpleMessage from other gossipers
	if we don't know the sender we add him to our list of peers
	we send this message to our peers except the sender
*/
func (g *Gossiper) handleSimplePacketG(pkt *SimpleMessage, sender *net.UDPAddr) {
	sender_formatted := ParseIPStr(sender)
	fmt.Println("SIMPLE MESSAGE origin " + pkt.OriginalName + " from " +
		sender_formatted + " contents " + pkt.Contents)

	newPkt := GossipPacket{Simple: &SimpleMessage{
		OriginalName:  pkt.OriginalName,
		RelayPeerAddr: ParseIPStr(g.udp_address),
		Contents:      pkt.Contents,
	}}

	//SEND PKT TO ALL
	packet_encoded, err := protobuf.Encode(&newPkt)
	if err != nil {
		fmt.Println("cannot encode message")
		log.Fatal(err)
	}
	g.listAllKnownPeers()
	mutex.Lock()
	for k := range g.set_of_peers {
		if k != sender_formatted {
			dst, err := net.ResolveUDPAddr("udp4", k)
			if err != nil {
				fmt.Println("cannot resolve addr of other gossiper")
				log.Fatal(err)
			}
			g.conn.WriteToUDP(packet_encoded, dst)
		}
	}
	mutex.Unlock()
}

func (g *Gossiper) sendRouteRumor(dst *net.UDPAddr) {

	newID := uint32(1)
	var index int
	isFirstMessage := true
	mutex.Lock()
	for i := 0; i < len(me.archives); i++ {
		if me.archives[i].Identifier == g.Name {
			newID = uint32(len(me.archives[i].msgs) + 1)
			index = i
			isFirstMessage = false
		}
	}

	if isFirstMessage {
		a := make(map[uint32]*RumorMessage)
		a[uint32(1)] = &RumorMessage{
			Origin: g.Name,
			ID:     newID,
			Text:   "",
		}
		g.vector_clock = append(g.vector_clock, PeerStatus{
			Identifier: g.Name,
			NextID:     uint32(2),
		})

		g.archives = append(g.archives, PeerMessage{
			Identifier: g.Name,
			msgs:       a,
		})
	} else {
		me.archives[index].msgs[newID] = &RumorMessage{
			Origin: g.Name,
			ID:     newID,
			Text:   "",
		}
		me.vector_clock[index].NextID += 1
	}
	mutex.Unlock()

	pktNew := GossipPacket{Rumor: &RumorMessage{
		Origin: g.Name,
		ID:     newID,
		Text:   "",
	}}

	pktByte, err := protobuf.Encode(&pktNew)

	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}

	mutex.Lock()
	g.conn.WriteToUDP(pktByte, dst)
	mutex.Unlock()
}

/**
This function receive all packet on the gossiper interface.
it also implement the anti-entropy process
	if the sending peer is unknonw we add him to our list of peers
	if the packet is a SimpleMessage we call handleSimplePacketG
	if the packet is a StatusPacket we call StatusPacketRoutine
	if the packet is a RumorMessage we call rumorMessageRoutine
*/
func (g *Gossiper) receiveMessageFromGossiper() {
	receivedInTime := false
	tickerAEntropy := time.NewTicker(time.Duration(ANTI_ENTROPY_TIMER) * time.Second)
	go func() {
		for _ = range tickerAEntropy.C {
			if !receivedInTime && len(g.set_of_peers) > 0 {
				dst := g.chooseRandomPeer()
				g.sendMyStatus(dst, 0)
			}
		}
	}()

	if g.rtimer > 0 {
		tickerRouting := time.NewTicker(time.Duration(g.rtimer) * time.Second)
		go func() {
			for _ = range tickerRouting.C {
				if len(g.set_of_peers) > 0 {
					dst := g.chooseRandomPeer()
					g.sendRouteRumor(dst)
				}
			}
		}()
	}

	defer g.conn.Close()
	for {
		var pkt GossipPacket = GossipPacket{}

		res := UDPPacket{}
		b := make([]byte, UDP_PACKET_SIZE)
		receivedInTime = false
		nb_byte_written, sender, err := g.conn.ReadFromUDP(b)
		receivedInTime = true
		res = UDPPacket{
			bytes:   b,
			err:     err,
			sender:  sender,
			nb_byte: nb_byte_written,
		}

		protobuf.Decode(res.bytes, &pkt)

		if res.nb_byte > 0 && res.err == nil {
			sender_formatted := ParseIPStr(res.sender)
			mutex.Lock()
			_, ok := g.set_of_peers[sender_formatted]
			if !ok && sender_formatted != ParseIPStr(g.udp_address) {
				g.set_of_peers[sender_formatted] = true
			}
			mutex.Unlock()
			if pkt.Simple != nil {
				g.handleSimplePacketG(pkt.Simple, res.sender)
			} else if pkt.Status != nil {
				g.StatusPacketRoutine(pkt.Status, res.sender)
			} else if pkt.Rumor != nil {
				if (sender_formatted != ParseIPStr(g.udp_address)) && (pkt.Rumor.Text != "") {
					printRumorMessageRcv(pkt.Rumor, res.sender)
					g.listAllKnownPeers()
				}
				g.rumorMessageRoutine(pkt.Rumor, res.sender)
			} else if pkt.Private != nil {
				g.privateMessageRoutine(pkt.Private)
			}
		}
	}
	return
}

/**
This function update both archives and vector clock with the arriving RumorMessage
it return two booleans
alreadyHave: true if we already have the pkt in our archives
notAdded: true if the packet has an id too high so we miss RumorMessage before that one
*/
func (g *Gossiper) updateArchivesVC(pkt *RumorMessage, sender *net.UDPAddr) (bool, bool) {
	var containSender bool = false
	var alreadyHave bool = false
	var notAdded bool = false
	mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		var ps PeerStatus = g.vector_clock[i]
		if ps.Identifier == pkt.Origin {
			if ps.NextID == pkt.ID {
				g.archives[i].msgs[ps.NextID] = &RumorMessage{
					Origin: pkt.Origin,
					ID:     pkt.ID,
					Text:   pkt.Text,
				}
				g.vector_clock[i].NextID = ps.NextID + 1
			} else if ps.NextID > pkt.ID {
				alreadyHave = true
			} else {
				notAdded = true
			}
			containSender = true
		}
	}

	if !containSender {
		a := make(map[uint32]*RumorMessage)
		var nID uint32
		if pkt.ID == 1 {
			nID = 2
			a[uint32(1)] = &RumorMessage{
				Origin: pkt.Origin,
				ID:     uint32(1),
				Text:   pkt.Text,
			}
		} else {
			nID = 1
			notAdded = true
		}
		g.vector_clock = append(g.vector_clock, PeerStatus{
			Identifier: pkt.Origin,
			NextID:     nID,
		})

		g.archives = append(g.archives, PeerMessage{
			Identifier: pkt.Origin,
			msgs:       a,
		})
	}
	fmt.Println("hi")
	if !alreadyHave && ParseIPStr(sender) != ParseIPStr(g.udp_address) {
		g.DSDV[pkt.Origin] = ParseIPStr(sender)
		fmt.Println("DSDV " + pkt.Origin + " " + ParseIPStr(sender))
	}

	mutex.Unlock()
	return alreadyHave, notAdded
}

func (g *Gossiper) privateMessageRoutine(pkt *PrivateMessage) {
	fmt.Println(" private message routine")
	mutex.Lock()
	name := g.Name
	mutex.Unlock()
	if pkt.Destination == name {
		_, ok := g.archives_private[pkt.Origin]
		if !ok {
			var new_array []PrivateMessage
			g.archives_private[pkt.Origin] = new_array
		}
		printPrivateMessageRcv(pkt)
		g.archives_private[pkt.Origin] = append(g.archives_private[pkt.Origin], *pkt)
	} else {
		mutex.Lock()
		dst, ok := g.DSDV[pkt.Destination]
		mutex.Unlock()
		if !ok {
			fmt.Println("Cannot forward message because destination unknown")
		} else {
			newPkt := GossipPacket{Private: &PrivateMessage{
				Origin:      pkt.Origin,
				ID:          0,
				Text:        pkt.Text,
				Destination: pkt.Destination,
				HopLimit:    pkt.HopLimit - 1,
			}}
			if newPkt.Private.HopLimit > 0 {
				pktByte, err := protobuf.Encode(&newPkt)
				if err != nil {
					fmt.Println("Encode of the packet failed")
					log.Fatal(err)
				}
				mutex.Lock()
				g.conn.WriteToUDP(pktByte, ParseStrIP(dst))
				mutex.Unlock()
			}
		}
	}
}

/**
This function is called to answer correctly to the received RumorMessage
we send our Status to the sender of the RumorMessage only if the sender is not ourselves
if the message interested us we monger it.
*/
func (g *Gossiper) rumorMessageRoutine(pkt *RumorMessage, sender *net.UDPAddr) {

	alreadyHave, notAdded := g.updateArchivesVC(pkt, sender)
	if ParseIPStr(sender) != ParseIPStr(g.udp_address) {
		g.sendMyStatus(sender, 0)
	}

	if !alreadyHave && !notAdded && len(g.set_of_peers) > 0 {
		newDst := g.chooseRandomPeer()
		g.sendingRoutine(pkt, newDst)
	}

	return
}

/**
This function return the UDPAddr of a peer chose randomly
*/
func (g *Gossiper) chooseRandomPeer() *net.UDPAddr {
	mutex.Lock()
	r := rand.Intn(len(g.set_of_peers))
	i := 0
	var addr *net.UDPAddr
	for k := range g.set_of_peers {
		if i == r {
			addr = ParseStrIP(k)
		}
		i++
	}
	mutex.Unlock()
	return addr
}

/**
This function send the rumor message to peer dst and wait for an ack from him
if we receive an ack we send the rumor to another random peer
*/
func (g *Gossiper) sendingRoutine(pkt *RumorMessage, dst *net.UDPAddr) {

	pktNew := GossipPacket{Rumor: &RumorMessage{
		Origin: pkt.Origin,
		ID:     pkt.ID,
		Text:   pkt.Text,
	}}
	pktByte, err := protobuf.Encode(&pktNew)

	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}
	if pkt.Text != "" {
		printRumorMessageSnd(dst)
	}

	mutex.Lock()
	g.conn.WriteToUDP(pktByte, dst)
	mutex.Unlock()
	/**
	rnd := rand.Int() % 2
	if rnd == 0 {
			newDst := g.chooseRandomPeer()
			flippedCoin(newDst)
			g.sendingRoutine(pkt, newDst)
		}*/

	ch := make(chan *StatusPacket, 1)
	go receivePkt(ch, dst, g.conn)
	select {
	case res := <-ch:
		g.StatusPacketRoutine(res, dst)
	case <-time.After(time.Duration(TIMEOUT_TIMER) * time.Second):
		rnd := rand.Int() % 2
		if rnd == 0 && len(g.set_of_peers) > 0 {
			newDst := g.chooseRandomPeer()
			flippedCoin(newDst)
			g.sendingRoutine(pkt, newDst)
		}
	}

	return
}

func receivePkt(out chan<- *StatusPacket, senderExpected *net.UDPAddr, conn *net.UDPConn) {
	pkt := GossipPacket{}
	b := make([]byte, UDP_PACKET_SIZE)
	nb_byte_written, sender, err := conn.ReadFromUDP(b)
	if nb_byte_written > 0 && err == nil {
		protobuf.Decode(b, &pkt)
		if pkt.Status != nil {
			if ParseIPStr(sender) == ParseIPStr(senderExpected) {
				out <- pkt.Status
			}
		} // BE CAREFULL HERE WE SOMETIME THROW AWAY RUMOR PACKET
	}
	return
}

/**
Function called when we receive a StatusPacket from another peer
numberToAsk represent the number of message that we don't already have
if numberToAsk > 0 then we send our status to the sender
if sender missed packet we send the Rumor message missed
*/
func (g *Gossiper) StatusPacketRoutine(pkt *StatusPacket, sender *net.UDPAddr) {
	printStatusMessageRcv(pkt, sender)
	g.listAllKnownPeers()
	vco := pkt.Want
	var vcm []PeerStatus
	mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		p := PeerStatus{
			Identifier: g.vector_clock[i].Identifier,
			NextID:     g.vector_clock[i].NextID,
		}
		vcm = append(vcm, p)
	}
	mutex.Unlock()
	//vcm := g.vector_clock

	numberToAsk := 0

	for i := 0; i < len(vcm); i++ {
		var contains bool = false
		elemM := vcm[i]
		for j := 0; j < len(vco); j++ {
			elemO := vco[j]
			if elemM.Identifier == elemO.Identifier {
				contains = true
				if elemM.NextID > elemO.NextID {
					g.sendUpdate(elemO, sender)
				} else if elemM.NextID < elemO.NextID {
					numberToAsk += int(elemO.NextID) - int(elemM.NextID)
				}
			}
		}
		if !contains {
			p := PeerStatus{
				Identifier: elemM.Identifier,
				NextID:     1,
			}
			g.sendUpdate(p, sender)
		}
	}

	for i := 0; i < len(vco); i++ {
		elemO := vco[i]
		var contains bool = false
		for j := 0; j < len(vcm); j++ {
			elemM := vcm[j]
			if elemM.Identifier == elemO.Identifier {
				contains = true
			}
		}
		if !contains {
			numberToAsk += int(elemO.NextID) - 1
		}
	}
	if numberToAsk > 0 {
		g.sendMyStatus(sender, numberToAsk)
	} else {
		fmt.Println("IN SYNC WITH " + ParseIPStr(sender))
	}
	return
}

/**
Send our status and wait for numberToAsk messages to arrive so we are up to date
*/
func (g *Gossiper) sendMyStatus(sender *net.UDPAddr, numberToAsk int) {
	var w []PeerStatus
	mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		w = append(w, PeerStatus{
			Identifier: g.vector_clock[i].Identifier,
			NextID:     g.vector_clock[i].NextID,
		})
	}
	mutex.Unlock()
	newPkt := GossipPacket{Status: &StatusPacket{
		Want: w,
	}}

	pktByte, err := protobuf.Encode(&newPkt)
	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}

	mutex.Lock()
	g.conn.WriteToUDP(pktByte, sender)
	mutex.Unlock()

	for i := 0; i < numberToAsk; i++ {
		pkt := GossipPacket{}
		b := make([]byte, UDP_PACKET_SIZE)
		nb_byte_written, sender, err := g.conn.ReadFromUDP(b)
		if nb_byte_written > 0 && err == nil {
			protobuf.Decode(b, &pkt)
			if pkt.Rumor != nil {
				if pkt.Rumor.Text != "" {
					printRumorMessageRcv(pkt.Rumor, sender)
					g.listAllKnownPeers()
				}
				g.updateArchivesVC(pkt.Rumor, sender)
			}
		}
	}
}

/**
We send the rumor messages from a specific origin the sender doesn't have yet
*/
func (g *Gossiper) sendUpdate(pDefault PeerStatus, sender *net.UDPAddr) {

	mutex.Lock()
	my_message := g.archives
	for i := 0; i < len(my_message); i++ {
		if my_message[i].Identifier == pDefault.Identifier {
			var finished bool = false
			for j := pDefault.NextID; !finished; {
				rm, ok := my_message[i].msgs[j]
				if ok {

					newPkt := GossipPacket{Rumor: &RumorMessage{
						Origin: rm.Origin,
						ID:     rm.ID,
						Text:   rm.Text,
					}}

					pktByte, err := protobuf.Encode(&newPkt)
					if err != nil {
						fmt.Println("Encode of the packet failed")
						log.Fatal(err)
					}
					g.conn.WriteToUDP(pktByte, sender)
					j++
				} else {
					finished = true
				}
			}
		}
	}
	mutex.Unlock()
	return
}
