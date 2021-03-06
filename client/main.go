package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"github.com/dedis/protobuf"
)

type ClientPacket struct {
	Simple  *SimpleMessage
	Private *PrivateMessage
	File 	*FileMessage
	Search	*SearchRequest
}

type FileMessage struct {
	Destination string
	Filename 	string
	Request		string
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

type SearchRequest struct {
	Origin 		string
	Budget 		uint64
	Keywords 	[]string
}


func main() {
	gossiperAddr := "127.0.0.1"
	uiport := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	dest := flag.String("dest", "", "destination for the private message")
	file := flag.String("file","","file to be indexed by the gossiper, or filename of the requested file")
	msg := flag.String("msg", "", "message to be send")
	req := flag.String("request","","request a chunk or metafile of this hash")
	clientport := flag.String("ClientPort", "10010", "port for the client to communicate with gossiper (default \"10010\")")
	keywords := flag.String("keywords","","filename keywords the client wants to search")
	budget := flag.Int("budget",0,"Optional parameter to specify the ring of search")

	flag.Parse()
	var pkt_to_enc ClientPacket
	if *dest != "" && *msg != "" {
		pkt_to_enc = ClientPacket{Private: &PrivateMessage{
			Origin:      "",
			ID:          0,
			Text:        *msg,
			Destination: *dest,
			HopLimit:    0,
		}}
		fmt.Println("send private")
	} else if *file != "" {
		pkt_to_enc = ClientPacket{File: &FileMessage{
			Destination: 	*dest,
			Filename:		*file,
			Request:		*req,
		}}
	}else if *keywords != ""{
		keywords_tab := strings.Split(*keywords, ",")
		pkt_to_enc = ClientPacket{Search: &SearchRequest{
			Origin: 		"",
			Budget:			uint64(*budget),
			Keywords:		keywords_tab,
		}}
		fmt.Println("Sending request")
		fmt.Println(keywords_tab)
	}else {
		pkt_to_enc = ClientPacket{Simple: &SimpleMessage{
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
