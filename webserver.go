package main

import (
	"encoding/json"
	"net/http"
	"fmt"
	"github.com/dedis/protobuf"
)

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "GET" {
		var values []RumorMessage
		mutex.Lock()
		for i := 0; i < len(me.archives); i++ {
			finished := false
			j := 1
			for !finished {
				elem, ok := me.archives[i].msgs[uint32(j)]
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
		mutex.Unlock()
		jsonValue, _ := json.Marshal(values)
		w.WriteHeader(http.StatusOK)
		w.Write(jsonValue)
		
	} else if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}

		pkt_to_enc := GossipPacket{Simple: &SimpleMessage{
			OriginalName:  "client",
			RelayPeerAddr: "",
			Contents:      r.Form.Get("value"),
		}}
		packetBytes, err := protobuf.Encode(&pkt_to_enc)
		me.clientConn.WriteToUDP(packetBytes, me.clientAddr)
	}

}

func NodeHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "GET" {
		mutex.Lock()
		values := me.set_of_peers
		mutex.Unlock()
		jsonValue, _ := json.Marshal(values)
		w.WriteHeader(http.StatusOK)
		w.Write(jsonValue)
	} else if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}
		mutex.Lock()
		me.set_of_peers[r.Form.Get("value")] = true
		mutex.Unlock()
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
	mutex.Lock()
	keys := make([]string, 0, len(me.DSDV))
	for k := range me.DSDV {
        keys = append(keys, k)
    }
	mutex.Unlock()
	jsonValue, _ := json.Marshal(keys)
	w.Write(jsonValue)
}

func PrivateMessageHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	
	var peer string
	for k := range r.URL.Query() {
        peer = k
    }
	
	mutex.Lock()
	fmt.Println(me.archives_private)
	keys := make([]string, 0, len(me.archives_private[peer]))
	for _,k := range me.archives_private[peer] {
        keys = append(keys, k)
    }
	mutex.Unlock()
	jsonValue, _ := json.Marshal(keys)
	w.Write(jsonValue)
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
