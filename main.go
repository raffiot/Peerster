package main

import (
	"flag"
	"net/http"
	"strings"
	"sync"
)

var UDP_PACKET_SIZE = 2048
var ANTI_ENTROPY_TIMER = 1 //Second
var TIMEOUT_TIMER = 1      //Second
var HOP_LIMIT = 10
var me *Gossiper
var mutex sync.Mutex

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

	me = NewGossiper(*gossipAddr, *name, strings.Split(*peers, ","), *simple, client_ip+":"+*uiport, *rtimer)

	if *rtimer > 0 {
		dst := me.chooseRandomPeer()
		me.sendRouteRumor(dst)
	}

	go me.receiveMessageFromClient()
	if *server {
		go me.receiveMessageFromGossiper()
		http.HandleFunc("/message", MessageHandler)
		http.HandleFunc("/node", NodeHandler)
		http.HandleFunc("/id", IdHandler)

		if err := http.ListenAndServe("localhost:8080", nil); err != nil {
			panic(err)
		}
	} else {
		me.receiveMessageFromGossiper()
	}

}
