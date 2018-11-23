package main

import (
	"fmt"
	"log"
	"net"
	"github.com/dedis/protobuf"
)


/**
Routine that handle the messages that we receive from the client
at <127.0.0.1:ClientPort>
*/
func (g *Gossiper) receiveMessageFromClient() {

	defer g.clientConn.Close()
	
	b := make([]byte, 10000)
	for {
		
		nb_byte_written, _, err := g.clientConn.ReadFromUDP(b)
		
		if err != nil{
			fmt.Println("Error when receiving")
			log.Fatal(err)
		} else if nb_byte_written > 0 {
			bb := make([]byte,nb_byte_written)
			copy(bb,b)

			
			
			
			go func(bb []byte){
				var pkt ClientPacket = ClientPacket{}
				protobuf.Decode(bb, &pkt)
				if pkt.Simple != nil {
					g.handleSimplePacket(pkt.Simple)
				} else if pkt.Private != nil {
					g.private_packet_handler_client(pkt.Private)
				} else if pkt.File != nil {
					//TO BE COMPLETE case with no dest !!
					if pkt.File.Request == "" {
						g.loadFile(pkt.File.Filename)
					} else {
						g.requestFile(pkt.File)
					}
				} else if pkt.Search != nil {
					g.search_packet_handler(pkt.Search) //launch as subroutine?
				} else {
					fmt.Println("Error malformed client packet")
				}
			}(bb) // CHECK IF PARENTESIS WELL DONE
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

		for k := range g.set_of_peers {
			dst, err := net.ResolveUDPAddr("udp4", k)
			if err != nil {
				fmt.Println("cannot resolve addr of ather gossiper")
				log.Fatal(err)
			}
			mutex.Lock()
			g.conn.WriteToUDP(pktByte, dst)
			mutex.Unlock()
		}

	} else {
		var my_ID uint32 = 1

		for i := 0; i < len(g.vector_clock); i++ {
			if g.vector_clock[i].Identifier == g.Name {
				my_ID = g.vector_clock[i].NextID
			}
		}

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

func (g *Gossiper) private_packet_handler_client(pkt *PrivateMessage) {

	newPkt := GossipPacket{Private: &PrivateMessage{
		Origin:      g.Name,
		ID:          0,
		Text:        pkt.Text,
		Destination: pkt.Destination,
		HopLimit:    HOP_LIMIT,
		}}
	fmt.Println(pkt.Destination)
	next_hop, ok := g.DSDV[pkt.Destination]

	if ok {
		_, ok := g.archives_private[pkt.Destination]
		if !ok {
			var new_array []PrivateMessage
				g.archives_private[pkt.Destination] = new_array
			}
			mutex.Lock()
			g.archives_private[pkt.Destination] = append(g.archives_private[pkt.Destination], *newPkt.Private)
			mutex.Unlock()	
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
}

func (g *Gossiper) search_packet_handler(pkt *SearchRequest){

	var sm []SearchMatch
	var ch chan bool
	mutex.Lock()
	g.pending_search.Is_pending = true
	g.pending_search.Nb_match = 0
	if pkt.Budget == 0 {
		ch = make(chan bool)
	}
	g.pending_search.ch = ch
	g.search_matches = sm //Clear old search_matches
	mutex.Unlock()
	if pkt.Budget == 0 {
		go g.search_routine(pkt)
	} else {
		// May be we want to send to all peers with same budget?	
		g.propagate_search(pkt)
	
	}
}
