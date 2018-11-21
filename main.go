package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dedis/protobuf"
)

var UDP_PACKET_SIZE = 2048
var ANTI_ENTROPY_TIMER = 2 //Second
var TIMEOUT_TIMER = 2      //Second
var TIMEOUT_FILE = 5       //Second
var HOP_LIMIT = uint32(10)
var me *Gossiper
var mutex sync.Mutex

func (g *Gossiper) gossiper_handler() {
	b := make([]byte, 10000)
	defer g.conn.Close()
	for {

		nb_byte_written, sender, err := g.conn.ReadFromUDP(b)

		if err != nil {
			fmt.Println("Error when receiving")
			log.Fatal(err)
		} else if nb_byte_written > 0 {

			bb := make([]byte, nb_byte_written)
			copy(bb, b)

			go func(bb []byte) {

				//Decode
				//bytes_packet := bb[:nb_byte_written]
				var pkt GossipPacket = GossipPacket{}
				protobuf.Decode(bb, &pkt)

				//Append sender to set of peers if unknown
				sender_formatted := ParseIPStr(sender)

				_, ok := g.set_of_peers[sender_formatted]
				if !ok && sender_formatted != ParseIPStr(g.udp_address) {
					mutex.Lock()
					g.set_of_peers[sender_formatted] = true
					mutex.Unlock()
				}

				//Process packet
				if pkt.Simple != nil {

					g.handleSimplePacketG(pkt.Simple, sender)
				} else if pkt.Status != nil {

					g.StatusPacketRoutine(pkt.Status, sender)
				} else if pkt.Rumor != nil {

					if (sender_formatted != ParseIPStr(g.udp_address)) && (pkt.Rumor.Text != "") {
						printRumorMessageRcv(pkt.Rumor, sender)
						g.listAllKnownPeers()
					}
					g.rumorMessageRoutine(pkt.Rumor, sender)
				} else if pkt.Private != nil {
					g.privateMessageRoutine(pkt.Private)
				} else if pkt.DataRequest != nil {
					if pkt.DataRequest.Destination == g.Name {
						g.receive_file_request_for_me(pkt.DataRequest)
					} else {
						g.forward_data_msg(&pkt)
					}
				} else if pkt.DataReply != nil {
					if pkt.DataReply.Destination == g.Name {
						g.receive_file_reply_for_me(pkt.DataReply)
					} else {
						g.forward_data_msg(&pkt)
					}
				} else {
					fmt.Println("Error malformed gossip packet")
				}
			}(bb)
		}
	}
}

func (g *Gossiper) rtimer_handler() {
	//first advisoring
	if len(g.set_of_peers) > 0 {
		dst := me.chooseRandomPeer()
		me.sendRouteRumor(dst)
	}

	//After a rtimer send route rumor to a random peer
	tickerRouting := time.NewTicker(time.Duration(g.rtimer) * time.Second)
	for _ = range tickerRouting.C {

		if len(g.set_of_peers) > 0 {
			dst := g.chooseRandomPeer()
			g.sendRouteRumor(dst)
		}
	}
}

func (g *Gossiper) anti_entropy_handler() {

	//After ANTI_ENTROPY_TIMER send a status to a peer
	tickerAEntropy := time.NewTicker(time.Duration(ANTI_ENTROPY_TIMER) * time.Second)
	for _ = range tickerAEntropy.C {
		//!receivedInTime &&

		if len(g.set_of_peers) > 0 {
			dst := g.chooseRandomPeer()
			g.sendMyStatus(dst, 0)
		}

	}
}

func main() {
	client_ip := "127.0.0.1"
	uiport := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper (default \"127.0.0.1:5000\")")
	name := flag.String("name", "", "name of the gossiper")
	peers := flag.String("peers", "", "coma separated list of peers of the form ip:port")
	rtimer := flag.Int("rtimer", 0, "route rumor sending in seconds, 0 to disable sending of route rumors (default 0)")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	server := flag.Bool("server", false, " run the server")
	flag.Parse()

	var peers_tab []string
	if *peers != "" {
		peers_tab = strings.Split(*peers, ",")
	}

	me = NewGossiper(*gossipAddr, *name, peers_tab, *simple, client_ip+":"+*uiport, *rtimer)

	if *rtimer > 0 {
		go me.rtimer_handler()
	}

	go me.anti_entropy_handler()

	go me.receiveMessageFromClient()
	if *server {
		go me.gossiper_handler()
		http.HandleFunc("/message", MessageHandler)
		http.HandleFunc("/node", NodeHandler)
		http.HandleFunc("/id", IdHandler)
		http.HandleFunc("/peer", PeerHandler)
		http.HandleFunc("/private", PrivateMessageHandler)
		http.HandleFunc("/file", FileMessageHandler)

		if err := http.ListenAndServe("localhost:8080", nil); err != nil {
			panic(err)
		}
	} else {
		me.gossiper_handler()
	}

}
