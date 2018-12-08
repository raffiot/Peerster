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
This function is called to answer correctly to the received RumorMessage
we send our Status to the sender of the RumorMessage only if the sender is not ourselves
if the message interested us we monger it.
*/
func (g *Gossiper) rumorMessageRoutine(pkt *RumorMessage, sender *net.UDPAddr) {

	alreadyHave, notAdded := g.updateArchivesVC(pkt, sender)
	if ParseIPStr(sender) != ParseIPStr(g.udp_address) {
		g.sendMyStatus(sender, 0)
	}

	if !alreadyHave && !notAdded && len(g.set_of_peers.set) > 0 {
		newDst := g.chooseRandomPeer()
		g.sendingRoutine(pkt, newDst)
	}

	return
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

	go g.ackRumorHandler(pkt, dst)

	return
}

func (g *Gossiper) sendRouteRumor(dst *net.UDPAddr) {

	newID := uint32(1)
	var index int
	isFirstMessage := true

	g.rumor_state.m.RLock()
	for i := 0; i < len(g.rumor_state.archives); i++ {
		if g.rumor_state.archives[i].Identifier == g.Name {
			newID = uint32(len(g.rumor_state.archives[i].msgs) + 1)
			index = i
			isFirstMessage = false
		}
	}
	g.rumor_state.m.RUnlock()

	if isFirstMessage {
		a := make(map[uint32]*RumorMessage)
		a[uint32(1)] = &RumorMessage{
			Origin: g.Name,
			ID:     newID,
			Text:   "",
		}
		g.rumor_state.m.Lock()
		g.rumor_state.vector_clock = append(g.rumor_state.vector_clock, PeerStatus{
			Identifier: g.Name,
			NextID:     uint32(2),
		})

		g.rumor_state.archives = append(g.rumor_state.archives, PeerMessage{
			Identifier: g.Name,
			msgs:       a,
		})
		g.rumor_state.m.Unlock()
	} else {
		g.rumor_state.m.Lock()
		g.rumor_state.archives[index].msgs[newID] = &RumorMessage{
			Origin: g.Name,
			ID:     newID,
			Text:   "",
		}
		g.rumor_state.vector_clock[index].NextID += 1
		g.rumor_state.m.Unlock()
	}

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
This function update both archives and vector clock with the arriving RumorMessage
it return two booleans
alreadyHave: true if we already have the pkt in our archives
notAdded: true if the packet has an id too high so we miss RumorMessage before that one
*/
func (g *Gossiper) updateArchivesVC(pkt *RumorMessage, sender *net.UDPAddr) (bool, bool) {
	var containSender bool = false
	var alreadyHave bool = false
	var notAdded bool = false

	for i := 0; i < len(g.rumor_state.vector_clock); i++ {
		var ps PeerStatus = g.rumor_state.vector_clock[i]
		if ps.Identifier == pkt.Origin {
			if ps.NextID == pkt.ID {
				g.rumor_state.m.Lock()
				g.rumor_state.archives[i].msgs[ps.NextID] = &RumorMessage{
					Origin: pkt.Origin,
					ID:     pkt.ID,
					Text:   pkt.Text,
				}
				g.rumor_state.vector_clock[i].NextID = ps.NextID + 1
				g.rumor_state.m.Unlock()
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
		g.rumor_state.m.Lock()
		g.rumor_state.vector_clock = append(g.rumor_state.vector_clock, PeerStatus{
			Identifier: pkt.Origin,
			NextID:     nID,
		})

		g.rumor_state.archives = append(g.rumor_state.archives, PeerMessage{
			Identifier: pkt.Origin,
			msgs:       a,
		})
		g.rumor_state.m.Unlock()
	}

	if !alreadyHave && ParseIPStr(sender) != ParseIPStr(g.udp_address) {
		g.dsdv.m.Lock()
		g.dsdv.state[pkt.Origin] = ParseIPStr(sender)
		g.dsdv.m.Unlock()
		fmt.Println("DSDV " + pkt.Origin + " " + ParseIPStr(sender))
	}

	return alreadyHave, notAdded
}

func (g *Gossiper) ackRumorHandler(pkt *RumorMessage, sender *net.UDPAddr) {

	chann := make(chan bool)

	ar := AckRumor{
		Identifier: pkt.Origin,
		NextID:     pkt.ID + 1,
		ch:         chann,
	}

	g.rumor_acks.m.Lock()
	g.rumor_acks.racks[ParseIPStr(sender)] = append(g.rumor_acks.racks[ParseIPStr(sender)], ar)
	g.rumor_acks.m.Unlock()
	select {
	case _ = <-chann:
		_ = <-chann

		//IF two time channel send bool it means that we are in sync

		index := -1
		g.rumor_acks.m.Lock()
		for i, v := range g.rumor_acks.racks[ParseIPStr(sender)] {
			if v.Identifier == pkt.Origin && v.NextID == pkt.ID+1 {
				index = i
			}
		}

		if index < 0 {
			fmt.Println("Error, ack entry not found")
			g.rumor_acks.m.Unlock()
			return
		}

		g.rumor_acks.racks[ParseIPStr(sender)] = append(g.rumor_acks.racks[ParseIPStr(sender)][:index], g.rumor_acks.racks[ParseIPStr(sender)][index+1:]...)
		g.rumor_acks.m.Unlock()

		rnd := rand.Int() % 2
		if rnd == 0 && len(g.set_of_peers.set) > 0 {
			newDst := g.chooseRandomPeer()
			flippedCoin(newDst)
			g.sendingRoutine(pkt, newDst)
		}
		fmt.Println("no timeout!")
	case <-time.After(time.Duration(TIMEOUT_TIMER) * time.Second):
		fmt.Println("timeout!")
		index := -1
		g.rumor_acks.m.Lock()
		for i, v := range g.rumor_acks.racks[ParseIPStr(sender)] {
			if v.Identifier == pkt.Origin && v.NextID == pkt.ID+1 {
				index = i
			}
		}
		if index < 0 {
			fmt.Println("Error, ack entry not found")
			return
		}
		//Is it possible that delete fail if just one message?

		g.rumor_acks.racks[ParseIPStr(sender)] = append(g.rumor_acks.racks[ParseIPStr(sender)][:index], g.rumor_acks.racks[ParseIPStr(sender)][index+1:]...)
		g.rumor_acks.m.Unlock()
		rnd := rand.Int() % 2
		if rnd == 0 && len(g.set_of_peers.set) > 0 {
			newDst := g.chooseRandomPeer()
			flippedCoin(newDst)
			g.sendingRoutine(pkt, newDst)
		}
	}

}
