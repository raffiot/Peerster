package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/dedis/protobuf"
)

type GossipPacket struct {
	Simple *SimpleMessage
}

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

func main() {
	gossiperAddr := "127.0.0.1"
	uiport := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	msg := flag.String("msg", "", "message to be send")
	clientport := flag.String("ClientPort", "10010", "port for the client to communicate with gossiper (default \"10010\")")

	flag.Parse()

	pkt_to_enc := GossipPacket{&SimpleMessage{
		OriginalName:  "client",
		RelayPeerAddr: "",
		Contents:      *msg,
	}}
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
