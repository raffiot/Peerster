package main

import (
	"fmt"
	"log"
	"github.com/dedis/protobuf"
)


func (g *Gossiper) privateMessageRoutine(pkt *PrivateMessage) {
	fmt.Println(" private message routine")
	
	name := g.Name
	
	if pkt.Destination == name {
		_, ok := g.archives_private.archives[pkt.Origin]
		if !ok {
			var new_array []PrivateMessage
			g.archives_private.archives[pkt.Origin] = new_array
		}
		printPrivateMessageRcv(pkt)
		g.archives_private.m.Lock()
		g.archives_private.archives[pkt.Origin] = append(g.archives_private.archives[pkt.Origin], *pkt)
		g.archives_private.m.Unlock()
	} else {
		
		dst, ok := g.dsdv.state[pkt.Destination]

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
