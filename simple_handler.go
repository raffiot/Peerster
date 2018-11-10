package main

import (
	"fmt"
	"log"
	"net"
	"github.com/dedis/protobuf"
)

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
	
	for k := range g.set_of_peers {
		if k != sender_formatted {
			dst, err := net.ResolveUDPAddr("udp4", k)
			if err != nil {
				fmt.Println("cannot resolve addr of other gossiper")
				log.Fatal(err)
			}
			mutex.Lock()
			g.conn.WriteToUDP(packet_encoded, dst)
			mutex.Unlock()
		}
	}
	
}
