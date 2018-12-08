package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/dedis/protobuf"
)

func (g *Gossiper) loadFile(filename string) {

	if filename == "" {
		fmt.Println("error filename in empty")
		return
	}
	var r io.Reader
	r, err := os.Open("./_SharedFiles/" + filename)
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		os.Exit(1)
	}

	fi, e := os.Stat("./_SharedFiles/" + filename)
	if e != nil {
		fmt.Println(e)
		return
	}

	// get the size
	filesize := int(fi.Size())



	var metafile []byte

	file_size_rem := filesize
	//repeat read as long as there is data
	for file_size_rem > 0 {
		buf_size := 8192
		if file_size_rem < 8192 {
			buf_size = file_size_rem
		}
		buf := make([]byte, buf_size)

		_, err := r.Read(buf)

		if err != nil {
			fmt.Println(err)
			return
		}

		sha_256 := sha256.New()
		sha_256.Write(buf)
		hash := sha_256.Sum(nil)
		metafile = append(metafile, hash...)
		file_size_rem -= 8192

		err = ioutil.WriteFile("./._Chunks/"+hex.EncodeToString(hash), buf, 0644)
		if err != nil {
			fmt.Println("Error during write of hash")
			fmt.Println(err)
			return
		}

	}
	sha_256 := sha256.New()
	sha_256.Write(metafile)
	metahash := sha_256.Sum(nil)

	err = ioutil.WriteFile("./._Chunks/"+hex.EncodeToString(metahash), metafile, 0644)
	if err != nil {
		fmt.Println("Error during write of hash")
		fmt.Println(err)
		return
	}

	fmt.Println(filename)
	fmt.Println(filesize)
	fmt.Println(hex.EncodeToString(metafile))
	fmt.Println(hex.EncodeToString(metahash))
	g.newFileNotice(filename, filesize, metahash)
	return
}

func (g *Gossiper) receive_file_request_for_me(pkt *DataRequest) {

	// We send a nil data if we don't have file.
	fmt.Println("receive request")
	file_id := hex.EncodeToString(pkt.HashValue)

	var buf []byte
	if fi, err := os.Stat("./._Chunks/" + file_id); !os.IsNotExist(err) {
		fmt.Println("chunk found")
		buf = make([]byte, int(fi.Size()))
		r, err := os.Open("./._Chunks/" + file_id)
		if err != nil {
			fmt.Printf("error opening file: %v\n", err)
			os.Exit(1)
		}
		_, err = r.Read(buf)
		if err != nil {
			fmt.Println("Error durring reading of chunk")
			fmt.Println(err)
			return
		}

	}

	newPkt := GossipPacket{DataReply: &DataReply{
		Origin:      g.Name,
		Destination: pkt.Origin,
		HopLimit:    HOP_LIMIT,
		HashValue:   pkt.HashValue, //Check if correct with copy and everything
		Data:        buf,
	}}

	g.sendDataPacket(newPkt)

}

func (g *Gossiper) forward_data_msg(pkt *GossipPacket) {
	if pkt.DataRequest != nil || pkt.DataReply != nil {
		var next_hop string
		var ok bool

		if pkt.DataRequest != nil {
			next_hop, ok = g.dsdv.state[pkt.DataRequest.Destination]
		} else {
			next_hop, ok = g.dsdv.state[pkt.DataReply.Destination]
		}

		if ok {
			var newPkt GossipPacket
			var hop uint32
			if pkt.DataRequest != nil {
				hop = pkt.DataRequest.HopLimit - 1
				newPkt = GossipPacket{DataRequest: &DataRequest{
					Origin:      pkt.DataRequest.Origin,
					Destination: pkt.DataRequest.Destination,
					HopLimit:    pkt.DataRequest.HopLimit - 1,
					HashValue:   pkt.DataRequest.HashValue,
				}}
			} else {
				hop = pkt.DataReply.HopLimit - 1
				newPkt = GossipPacket{DataReply: &DataReply{
					Origin:      pkt.DataReply.Origin,
					Destination: pkt.DataReply.Destination,
					HopLimit:    pkt.DataReply.HopLimit - 1,
					HashValue:   pkt.DataReply.HashValue,
					Data:        pkt.DataReply.Data,
				}}
			}
			if hop > 0 {
				pktByte, err := protobuf.Encode(&newPkt)
				if err != nil {
					fmt.Println("Error when encoding")
					log.Fatal(err)
				}
				mutex.Lock()
				g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
				mutex.Unlock()
			} else {
				fmt.Println("HOP = 0")
			}
		} else {
			fmt.Println("destination unknown for data message")
		}
	}
}

func (g *Gossiper) requestFileFromSearch(pkt *FileMessage) {
	//Find match between pkt.Request and SearchMatches.sm[i]
	//Construct slice FromDests
	//Create File & AckFile
	//Begin go routine
	//Send request
	tmp, _ := hex.DecodeString(pkt.Request)
	var sm SearchMatch
	found := false	
	g.search_matches.m.RLock()
	for _,match := range g.search_matches.sm {
		if bytes.Equal(match.MetafileHash,tmp) {
			sm = match
			found = true
			break
		}
	}
	g.search_matches.m.RUnlock()
	if !found {
		fmt.Println("No search match this request")
		return
	}
	fmt.Println("matches dest")
	fmt.Println(sm.Matches)
	dests := make([]string,sm.ChunkCount)
	for k,v := range sm.Matches {
		for _,i := range v {
			// Because chunk map start from 1 to n and not 0 to n-1
			dests[i-1] = k
		}
	}

	chann := make(chan bool)
	ack := AckFile{
		hashExpected:	pkt.Request,
		dest:	dests[0],
		ch:		chann,		
	}

	f := &FileForSearch{
		Filename: pkt.Filename,
		Filesize: 0,
		Metafile: nil,
		Metahash: tmp,
		FromDests: dests,
		ack: ack,
	}
	g.file_pending.m.Lock()
	g.file_pending.pf[pkt.Request] = f
	g.file_pending.m.Unlock()

	go g.fileTimeout(&ack)
	
	if _, err := os.Stat("./_Downloads/" + pkt.Filename); !os.IsNotExist(err) {
		err = os.Remove("./_Downloads/" + pkt.Filename)
		if err != nil {
			fmt.Println("couldn't remove file")
			return
		}
	}

	newPkt := GossipPacket{DataRequest: &DataRequest{
		Origin:      g.Name,
		Destination: dests[0],
		HopLimit:    HOP_LIMIT,
		HashValue:   tmp, //Check if correct with copy and everything
	}}

	g.sendDataPacket(newPkt)

	downloadPrint(pkt.Filename, -2, dests[0])
}

func (g *Gossiper) requestFile(pkt *FileMessage) {

	tmp, _ := hex.DecodeString(pkt.Request)

	

	

	//_, ok := g.file_pending.pf[pkt.Destination]
	chann := make(chan bool)
	ack := AckFile{
		hashExpected:	pkt.Request,
		dest:	pkt.Destination,
		ch:		chann,		
	}

	f := &FileForSearch{
		Filename: pkt.Filename,
		Filesize: 0,
		Metafile: nil,
		Metahash: tmp,
		FromDests: nil,
		ack: ack,
	}
	g.file_pending.m.Lock()
	g.file_pending.pf[pkt.Request] = f
	g.file_pending.m.Unlock()
	
	go g.fileTimeout(&ack)
	
	if _, err := os.Stat("./_Downloads/" + pkt.Filename); !os.IsNotExist(err) {
		err = os.Remove("./_Downloads/" + pkt.Filename)
		if err != nil {
			fmt.Println("couldn't remove file")
			return
		}
	}

	newPkt := GossipPacket{DataRequest: &DataRequest{
		Origin:      g.Name,
		Destination: pkt.Destination,
		HopLimit:    HOP_LIMIT,
		HashValue:   tmp, //Check if correct with copy and everything
	}}

	g.sendDataPacket(newPkt)

	downloadPrint(pkt.Filename, -2, pkt.Destination)
}

func writeChunk(file_title string, data []byte){
	err := ioutil.WriteFile("./._Chunks/"+file_title, data, 0644)
	if err != nil {
		fmt.Println("Error during write of hash")
		fmt.Println(err)
		return
	}
}

func writeFile(filename string, data []byte){
	f, err := os.OpenFile("./_Downloads/"+filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}

func (g *Gossiper) receive_file_reply_for_me(pkt *DataReply) {
	

	var currentFile *FileForSearch 
	hash_received := hex.EncodeToString(pkt.HashValue)

	var file_metahash string

	g.file_pending.m.Lock()
	// Check if with map it don't give me a copy so modification on ack_file won't modify state
	for key, file := range g.file_pending.pf {
		if file.ack.hashExpected ==  hash_received {
			currentFile = file
			file_metahash = key
		}
	}
	g.file_pending.m.Unlock()

	if pkt.Data != nil && len(pkt.Data) > 0 && currentFile != nil{
		var newPkt *GossipPacket
		
		if currentFile.Metafile != nil{
			//We receive a chunk
			pkt_waited := currentFile.Filesize / 256
			if bytes.Equal(pkt.HashValue, currentFile.Metafile[pkt_waited:pkt_waited+32]){
				writeChunk(hash_received, pkt.Data)
				writeFile(currentFile.Filename,pkt.Data)
				if pkt_waited+32 >= len(currentFile.Metafile) {
					//It was the last packet we waited for this file
					downloadPrint(currentFile.Filename, (pkt_waited+32)/32, pkt.Origin)
					downloadPrint(currentFile.Filename, -1, "")
					g.file_pending.m.Lock()
					currentFile.ack.hashExpected = ""
					currentFile.ack.ch <- false

					// I withdraw current file from pending files map
					delete(g.file_pending.pf,file_metahash)
					g.file_pending.m.Unlock()

					
					
				} else {

					currentFile.Filesize += 8192

					pkt_wanted := currentFile.Filesize / 256
					downloadPrint(currentFile.Filename, pkt_wanted/32, pkt.Origin)
					newPkt = &GossipPacket{DataRequest: &DataRequest{
						Origin:      g.Name,
						Destination: currentFile.FromDests[pkt_wanted/32],
						HopLimit:    HOP_LIMIT,
						HashValue:   currentFile.Metafile[pkt_wanted : pkt_wanted+32],
					}}
					g.file_pending.m.Lock()
					currentFile.ack.hashExpected = hex.EncodeToString(currentFile.Metafile[pkt_wanted : pkt_wanted+32])
					currentFile.ack.dest = currentFile.FromDests[pkt_wanted/32]
					currentFile.ack.ch <- true
					g.file_pending.m.Unlock()
				}
			}			
		} else {
			//We receive metafile
			//I copy it and store in file_pending
			data_received := make([]byte, len(pkt.Data))
			copy(data_received, pkt.Data)
			nb_chunks := len(data_received)/32				
			g.file_pending.m.Lock()
			currentFile.Metafile = data_received

			//If we didn't know about this file from search we will ack all file to same node
			if currentFile.FromDests == nil {
				from_dest := make([]string,nb_chunks)
				for i:=0; i < nb_chunks; i++ {
					from_dest[i] = pkt.Origin
				}			
				currentFile.FromDests = from_dest
			}
			currentFile.ack.dest = currentFile.FromDests[0]
			currentFile.ack.hashExpected = hex.EncodeToString(currentFile.Metafile[0:32])
			currentFile.ack.ch <- true
			g.file_pending.m.Unlock()
				
			downloadPrint(currentFile.Filename, 0, pkt.Origin)
			newPkt = &GossipPacket{DataRequest: &DataRequest{
				Origin:      g.Name,
				Destination: currentFile.FromDests[0],
				HopLimit:    HOP_LIMIT,
				HashValue:   currentFile.Metafile[0:32],
			}}
				
			writeChunk(hash_received, pkt.Data)
		}
		if newPkt != nil {
			g.sendDataPacket(*newPkt)
		}
	} else if pkt.Data == nil && currentFile != nil {
		
		g.file_pending.m.Lock()
		currentFile.ack.hashExpected = ""
		currentFile.ack.ch <- false

		// I withdraw current file from pending files map
		delete(g.file_pending.pf,file_metahash)
		g.file_pending.m.Unlock()
	} else {
		fmt.Println("no match")
	}

}

func (g *Gossiper) fileTimeout(ack *AckFile) {

	res := true
	for res {
		select {
		case <-time.After(time.Duration(TIMEOUT_FILE) * time.Second):
			mutex.Lock()
			// Is af reference still up to date??
			tmp, _ := hex.DecodeString(ack.hashExpected)
			mutex.Unlock()
			newPkt := GossipPacket{DataRequest: &DataRequest{
				Origin:      g.Name,
				Destination: ack.dest,
				HopLimit:    HOP_LIMIT,
				HashValue:   tmp,
			}}
			g.sendDataPacket(newPkt)

		case res = <-ack.ch:

		}
	}

/**
	index := -1
	mutex.Lock()
	for i, v := range g.file_acks[dest] {
		if v.HashExpected == "" {
			index = i
		}
	}
	if index >= 0 {
		g.file_acks[dest] = append(g.file_acks[dest][:index], g.file_acks[dest][index+1:]...)
	}
	mutex.Unlock()*/
}

func (g *Gossiper) sendDataPacket(pkt GossipPacket) {
	pktByte, err := protobuf.Encode(&pkt)
	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}
	var dest string
	if pkt.DataReply != nil {
		dest = pkt.DataReply.Destination
	} else if pkt.DataRequest != nil {
		dest = pkt.DataRequest.Destination
	} else {
		fmt.Println("Error wrong use of sendDataPacket function")
		return
	}
	next_hop, ok := g.dsdv.state[dest]
	if ok {
		mutex.Lock()
		g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
		mutex.Unlock()
	} else {
		fmt.Println("destination unknown for data request message")
	}

}
