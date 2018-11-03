package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/dedis/protobuf"
)

type GossipPacket struct {
	Simple  *SimpleMessage
	Rumor   *RumorMessage
	Status  *StatusPacket
	Private *PrivateMessage
}

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

type PeerStatus struct {
	Identifier string
	NextID     uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

func main() {
	gossiperAddr := "127.0.0.1"
	uiport := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	dest := flag.String("dest", "", "destination for the private message")
	msg := flag.String("msg", "", "message to be send")
	clientport := flag.String("ClientPort", "10010", "port for the client to communicate with gossiper (default \"10010\")")

	flag.Parse()
	var pkt_to_enc GossipPacket
	if *dest != "" && *msg != "" {
		pkt_to_enc = GossipPacket{Private: &PrivateMessage{
			Origin:      "",
			ID:          0,
			Text:        *msg,
			Destination: *dest,
			HopLimit:    0,
		}}
		fmt.Println("send private")
	} else {
		pkt_to_enc = GossipPacket{Simple: &SimpleMessage{
			OriginalName:  "client",
			RelayPeerAddr: "",
			Contents:      *msg,
		}}
	}

	packetBytes, err := protobuf.Encode(&pkt_to_enc)

	if err != nil {
		fmt.Println("Encoding of message went wrong !")
		log.Fatal(err)
	}
	dst, err := net.ResolveUDPAddr("udp4", gossiperAddr+":"+*uiport)
	if err != nil {
		fmt.Println("resolve udp addr went wrong !")
		log.Fatal(err)
	}

	src, err := net.ResolveUDPAddr("udp4", gossiperAddr+":"+*clientport)
	conn, err := net.ListenUDP("udp4", src)

	if err != nil {
		fmt.Println("DialUDP when wrong !")
		log.Fatal(err)
	}

	conn.WriteToUDP(packetBytes, dst)

	if err != nil {
		fmt.Println("Write to udp when wrong !")
		log.Fatal(err)
	}

	defer conn.Close()
}
