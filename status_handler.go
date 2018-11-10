package main

import (
	"fmt"
	"log"
	"net"
	"github.com/dedis/protobuf"
)

/**
Function called when we receive a StatusPacket from another peer
numberToAsk represent the number of message that we don't already have
if numberToAsk > 0 then we send our status to the sender
if sender missed packet we send the Rumor message missed
*/
func (g *Gossiper) StatusPacketRoutine(pkt *StatusPacket, sender *net.UDPAddr) {
	printStatusMessageRcv(pkt, sender)
	g.listAllKnownPeers()

	cop := make([]AckRumor,len(g.rumor_acks[ParseIPStr(sender)]))
	copy(cop, g.rumor_acks[ParseIPStr(sender)])
	//mutex.Lock()
	if g.rumor_acks[ParseIPStr(sender)] != nil {
		
		for _,v1 := range pkt.Want{
			for i,v2 := range cop{
				if v1.Identifier == v2.Identifier && v1.NextID == v2.NextID {
					g.rumor_acks[ParseIPStr(sender)][i].ch<-true

				}
			}
		}
	}
	//mutex.Unlock()
	
	vco := pkt.Want
	var vcm []PeerStatus
	//mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		p := PeerStatus{
			Identifier: g.vector_clock[i].Identifier,
			NextID:     g.vector_clock[i].NextID,
		}
		vcm = append(vcm, p)
	}
	//mutex.Unlock()
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
		//mutex.Lock()
		if g.rumor_acks[ParseIPStr(sender)] != nil {
			for _,v1 := range pkt.Want{
				for i,v2 := range cop{
					if v1.Identifier == v2.Identifier && v1.NextID == v2.NextID {
						g.rumor_acks[ParseIPStr(sender)][i].ch<-true
					}
				}
			}
		}
		//mutex.Unlock()
	}
	return
}

/**
Send our status and wait for numberToAsk messages to arrive so we are up to date
*/
func (g *Gossiper) sendMyStatus(sender *net.UDPAddr, numberToAsk int) {
	var w []PeerStatus
	
	for i := 0; i < len(g.vector_clock); i++ {
		w = append(w, PeerStatus{
			Identifier: g.vector_clock[i].Identifier,
			NextID:     g.vector_clock[i].NextID,
		})
	}
	
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
	
	/**
	
	
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
	}*/
}

/**
We send the rumor messages from a specific origin the sender doesn't have yet
*/
func (g *Gossiper) sendUpdate(pDefault PeerStatus, sender *net.UDPAddr) {

	
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
					mutex.Lock()
					g.conn.WriteToUDP(pktByte, sender)
					mutex.Unlock()
					j++
				} else {
					finished = true
				}
			}
		}
	}
	
	return
}
