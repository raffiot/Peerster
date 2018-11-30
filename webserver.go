package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"log"
	"github.com/dedis/protobuf"
)

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "GET" {
		var values []RumorMessage

		me.rumor_state.m.Lock()
		my_message := make([]PeerMessage,len(me.rumor_state.archives))
		copy(my_message,me.rumor_state.archives)
		me.rumor_state.m.Unlock()

		for i := 0; i < len(my_message); i++ {
			finished := false
			j := 1
			for !finished {
				elem, ok := my_message[i].msgs[uint32(j)]
				j++
				if ok {
					if elem.Text != "" {
						values = append(values, *elem)
					}
				} else {
					finished = true
				}
			}
		}
		
		jsonValue, _ := json.Marshal(values)
		w.WriteHeader(http.StatusOK)
		w.Write(jsonValue)

	} else if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}

		pkt_to_enc := ClientPacket{Simple: &SimpleMessage{
			OriginalName:  "client",
			RelayPeerAddr: "",
			Contents:      r.Form.Get("value"),
		}}
		packetBytes, err := protobuf.Encode(&pkt_to_enc)
		mutex.Lock()
		me.clientConn.WriteToUDP(packetBytes, me.clientAddr)
		mutex.Unlock()
	}

}

func NodeHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "GET" {
		values := me.set_of_peers.set
		jsonValue, _ := json.Marshal(values)
		w.WriteHeader(http.StatusOK)
		w.Write(jsonValue)
	} else if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		me.set_of_peers.m.Lock()
		me.set_of_peers.set[r.Form.Get("value")] = true
		me.set_of_peers.m.Unlock()
	}
}

func IdHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	values := me.Name
	jsonValue, _ := json.Marshal(values)
	w.Write(jsonValue)
}

func PeerHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	if r.Method == "GET" {
		//mutex.Lock()
		keys := make([]string, 0, len(me.dsdv.state))
		for k := range me.dsdv.state {
			keys = append(keys, k)
		}
		//mutex.Unlock()
		jsonValue, _ := json.Marshal(keys)
		w.Write(jsonValue)
	}
}

func PrivateMessageHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	if r.Method == "GET" {
		var peer string
		for k := range r.URL.Query() {
			peer = k
		}

		
		keys := make([]PrivateMessage, 0, len(me.archives_private.archives[peer]))
		for _, k := range me.archives_private.archives[peer] {
			keys = append(keys, k)
		}
		
		jsonValue, _ := json.Marshal(keys)
		w.Write(jsonValue)
	} else if r.Method == "POST" {
		fmt.Println("I received post")
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		fmt.Println(r.Form.Get("destination"))
		pkt_to_enc := ClientPacket{Private: &PrivateMessage{
			Origin:      "",
			ID:          0,
			Text:        r.Form.Get("message"),
			Destination: r.Form.Get("destination"),
			HopLimit:    0,
		}}

		packetBytes, err := protobuf.Encode(&pkt_to_enc)
		fmt.Println("sending Private")
		mutex.Lock()
		me.clientConn.WriteToUDP(packetBytes, me.clientAddr)
		mutex.Unlock()
	}
}

func FileMessageHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	err := r.ParseForm()
	if err != nil {
		panic(err)
	}
	fmt.Println(r.Form.Get("type"))
	var pkt_to_enc ClientPacket
	type_post := r.Form.Get("type")
	if type_post == "upload" {
		pkt_to_enc = ClientPacket{File: &FileMessage{
			Destination: 	"",
			Filename:		r.Form.Get("filename"),
			Request:		"",
		}}	
	} else if type_post == "request" {
		pkt_to_enc = ClientPacket{File: &FileMessage{
			Destination: 	r.Form.Get("destination"),
			Filename:		r.Form.Get("filename"),
			Request:		r.Form.Get("request"),
		}}
		
	} else{
		fmt.Println("Unknown file request type")
		return
	}
	packetBytes, err := protobuf.Encode(&pkt_to_enc)

	if err != nil {
		fmt.Println("Encoding of message went wrong !")
		log.Fatal(err)
	}
	mutex.Lock()
	me.clientConn.WriteToUDP(packetBytes, me.clientAddr)
	mutex.Unlock()
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
