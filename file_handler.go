package main

import (
	"fmt"
	"os"
	"io"
	"io/ioutil"
	"log"
	"crypto/sha256"
	"encoding/hex"
	"bytes"
	"time"
	"github.com/dedis/protobuf"
)

func (g *Gossiper) loadFile(filename string){

	if filename == "" {
		fmt.Println("error filename in empty")
		return
	}
	var r io.Reader
	r, err := os.Open("./_SharedFiles/"+filename)
	if err != nil {
		fmt.Printf("error opening file: %v\n",err)
		os.Exit(1)
	}
	
	
	fi, e := os.Stat("./_SharedFiles/"+filename);
	if e != nil {
		fmt.Println(e)
		return 
	}
	
	
	// get the size
	filesize := int(fi.Size())
	//Create containers
	//sha_256 := sha256.New()
	
	//32*int(math.Ceil(float64(filesize)/float64(8)))
	
	var metafile []byte
	//chunks := make([][]byte,0,int(ceil(fileSize/8)))

	
	
	file_size_rem := filesize
	//repeat read as long as there is data
	for file_size_rem > 0 {
		buf_size := 8192
		if file_size_rem < 8192 {
			buf_size = file_size_rem 
		}
		buf := make([]byte,buf_size)
		
		_,err := r.Read(buf)

		if err != nil {
			fmt.Println(err)
			return 
		}
		//chunks = append(chunks,buf)
		
		sha_256 := sha256.New()
		sha_256.Write(buf)
		hash := sha_256.Sum(nil)
		metafile = append(metafile,hash...)
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

	return
}

func (g *Gossiper) receive_file_request_for_me(pkt *DataRequest){
	
	// We send a nil data if we don't have file.

	file_id := hex.EncodeToString(pkt.HashValue)

	var buf []byte
	if fi, err := os.Stat("./._Chunks/"+file_id); !os.IsNotExist(err) {
		buf = make([]byte,int(fi.Size()))
		r, err := os.Open("./._Chunks/"+file_id)
		if err != nil {
			fmt.Printf("error opening file: %v\n",err)
			os.Exit(1)
		}
		_,err = r.Read(buf)
		if err != nil {
			fmt.Println("Error durring reading of chunk")
			fmt.Println(err)
			return 
		}
	
		
	}
	
	
	newPkt := GossipPacket{DataReply: &DataReply{
		Origin: g.Name,
		Destination: pkt.Origin,
		HopLimit: HOP_LIMIT,
		HashValue: pkt.HashValue, //Check if correct with copy and everything
		Data: buf,
	}}
	
	
	g.sendDataPacket(newPkt)	
	
}

func (g *Gossiper) forward_data_msg(pkt *GossipPacket){
	if pkt.DataRequest != nil ||  pkt.DataReply != nil{
		var next_hop string
		var ok bool

		if pkt.DataRequest != nil{
			next_hop, ok = g.DSDV[pkt.DataRequest.Destination]
		} else{
			next_hop, ok = g.DSDV[pkt.DataReply.Destination]
		}
		

		if ok {
			var newPkt GossipPacket
			var hop uint32
			if pkt.DataRequest != nil{
				hop = pkt.DataRequest.HopLimit -1
				newPkt = GossipPacket{DataRequest: &DataRequest{
					Origin: pkt.DataRequest.Origin,
					Destination: pkt.DataRequest.Destination,
					HopLimit: pkt.DataRequest.HopLimit -1,
					HashValue: pkt.DataRequest.HashValue,
				}}
			} else{
				hop = pkt.DataRequest.HopLimit -1
				newPkt = GossipPacket{DataReply: &DataReply{
					Origin: pkt.DataReply.Origin,
					Destination: pkt.DataReply.Destination,
					HopLimit: pkt.DataReply.HopLimit -1,
					HashValue: pkt.DataReply.HashValue,
					Data: pkt.DataReply.Data,
				}}
			}
			if hop > 0 {
				pktByte, err := protobuf.Encode(&newPkt)
				if err != nil{
					fmt.Println("Error when encoding")
					log.Fatal(err)
				}
				mutex.Lock()
				g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
				mutex.Unlock()
			} else {
				fmt.Println("HOP = 0")
			}
		} else{
			fmt.Println("destination unknown for data message")
		}
	}
}


func (g *Gossiper) requestFile(pkt *FileMessage){
	
	tmp,_ := hex.DecodeString(pkt.Request)
	
	go g.fileTimeout(pkt.Destination,pkt.Request)
	
	newPkt := GossipPacket{DataRequest: &DataRequest{
		Origin: g.Name,
		Destination: pkt.Destination,
		HopLimit: HOP_LIMIT,
		HashValue: tmp, //Check if correct with copy and everything
	}}
	
	g.sendDataPacket(newPkt)
	
	
	_,ok := g.file_pending[pkt.Destination]
	f := File{
		Filename: 	pkt.Filename,
		Filesize:	0,
		Metafile:	nil,
		Metahash:	tmp,
	}
	if ok {
		mutex.Lock()
		g.file_pending[pkt.Destination] = append(g.file_pending[pkt.Destination], f)
		mutex.Unlock()
	} else {
		var new_array []File
		new_array = append(new_array,f)
		mutex.Lock()
		g.file_pending[pkt.Destination] = new_array
		mutex.Unlock()
	}
	
	if _, err := os.Stat("./_SharedFiles/"+pkt.Filename); !os.IsNotExist(err) {
		err = os.Remove("./_SharedFiles/"+pkt.Filename)
		if err != nil {
			fmt.Println("couldn't remove file")
			return
		}
	}

	downloadPrint(pkt.Filename,-2, pkt.Destination)
}

func (g *Gossiper) receive_file_reply_for_me(pkt *DataReply){
	var ack_file AckFile
	have_ack_file := false
	mutex.Lock()
	for i,_ := range g.file_acks[pkt.Origin]{
		if g.file_acks[pkt.Origin][i].HashExpected == hex.EncodeToString(pkt.HashValue){
			ack_file = g.file_acks[pkt.Origin][i]
			have_ack_file = true
		}
	}
	mutex.Unlock()
	
	if pkt.Data != nil {
		var newPkt GossipPacket

		mutex.Lock()
		pending_array := make([]File, len(g.file_pending[pkt.Origin]))
		copy(pending_array,g.file_pending[pkt.Origin]) 
		mutex.Unlock()

		index := -1
		found := false
		for i,_ := range pending_array {

			if bytes.Equal(pkt.HashValue, pending_array[i].Metahash){
	
				//We received first packet for a file
				arr := make([]byte,len(pkt.Data))
				copy(arr,pkt.Data)
				pending_array[i].Metafile = arr
				downloadPrint(pending_array[i].Filename,0, pkt.Origin)
				
				newPkt = GossipPacket{DataRequest: &DataRequest{
					Origin: g.Name,
					Destination: pkt.Origin,
					HopLimit: HOP_LIMIT,
					HashValue: pending_array[i].Metafile[0:32],
				}}
				
				if have_ack_file {
					mutex.Lock()
					ack_file.HashExpected = hex.EncodeToString(pending_array[i].Metafile[0:32])
					ack_file.ch<-true
					mutex.Unlock()
				}
				
				
				f, err := os.OpenFile("./._Chunks/"+hex.EncodeToString(pkt.HashValue), os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Println("Error when loading the file")
					log.Fatal(err)
					return
				}
				if _, err := f.Write(pkt.Data); err != nil {
					log.Fatal(err)
				}
				if err := f.Close(); err != nil {
					log.Fatal(err)
				}

				found = true
				break
			} else if pending_array[i].Metafile != nil {
		
				pkt_waited := pending_array[i].Filesize / 256

				//If the packet we receive is confirmed to be the next because the hash is the good one
				if bytes.Equal(pkt.HashValue, pending_array[i].Metafile[pkt_waited:pkt_waited+32]){

					found = true
					err := ioutil.WriteFile("./._Chunks/"+hex.EncodeToString(pkt.HashValue), pkt.Data, 0644)
					if err != nil {
						fmt.Println("Error during write of hash")
						fmt.Println(err)
						return
					}
					
					f, err := os.OpenFile("./_SharedFiles/"+pending_array[i].Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if _, err := f.Write(pkt.Data); err != nil {
						log.Fatal(err)
					}
					if err := f.Close(); err != nil {
						log.Fatal(err)
					}
					
					if pkt_waited+32 >= len(pending_array[i].Metafile){
						//It was the last packet we waited for this file
						downloadPrint(pending_array[i].Filename,(pkt_waited+32) / 32 , pkt.Origin)
						downloadPrint(pending_array[i].Filename,-1, "")
						index = i
						if have_ack_file {
							mutex.Lock()
							ack_file.HashExpected = ""
							ack_file.ch<-false
							mutex.Unlock()
						}
					} else{
						
						pending_array[i].Filesize += 8192

						pkt_wanted := pending_array[i].Filesize / 256
						downloadPrint(pending_array[i].Filename,pkt_wanted / 32 , pkt.Origin)
						newPkt = GossipPacket{DataRequest: &DataRequest{
							Origin: g.Name,
							Destination: pkt.Origin,
							HopLimit: HOP_LIMIT,
							HashValue: pending_array[i].Metafile[pkt_wanted:pkt_wanted+32],
						}}
						if have_ack_file {
							mutex.Lock()
							ack_file.HashExpected = hex.EncodeToString(pending_array[i].Metafile[pkt_wanted:pkt_wanted+32])
							ack_file.ch<-true
							mutex.Unlock()
						}
					}
					break					
				}
				
			}	
		}
		if index != -1 {
			// File download has been completed

			mutex.Lock()
			pending_array = append(pending_array[:index],pending_array[index+1:]...)
			mutex.Unlock()
			//this should modify the reference so its okay
			//g.file_pending[pkt.Origin] = append(g.file_pending[:index],g.file_pending[index+1:]...)
			
		} else if found {
			g.sendDataPacket(newPkt)			
		}
		mutex.Lock()
		g.file_pending[pkt.Origin] = pending_array
		mutex.Unlock()
	} else {
		ack_file.ch<-false
	}

}

func (g *Gossiper) fileTimeout(dest string, hash string){
	chann := make(chan bool)
	af := AckFile{
		HashExpected:	hash,
		ch:				chann,
	}
	mutex.Lock()
	g.file_acks[dest] = append(g.file_acks[dest],af)
	mutex.Unlock()
	
	res := true
	for res {
		select{
			case <-time.After(time.Duration(TIMEOUT_FILE) * time.Second):
				mutex.Lock()
				// Is af reference still up to date??
				tmp,_ := hex.DecodeString(af.HashExpected)
				mutex.Unlock()
				newPkt := GossipPacket{DataRequest: &DataRequest{
					Origin: g.Name,
					Destination: dest,
					HopLimit: HOP_LIMIT,
					HashValue: tmp,
				}}
				g.sendDataPacket(newPkt)

			case res=<-chann:
			
		}
	}
	index := -1
	mutex.Lock()
	for i,v := range g.file_acks[dest]{
		if v.HashExpected == ""{
			index = i
		}
	}
	if index >=0 {
		g.file_acks[dest] = append(g.file_acks[dest][:index],g.file_acks[dest][index+1:]...)
	}
	mutex.Unlock()
}

func (g *Gossiper) sendDataPacket(pkt GossipPacket){
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
	next_hop, ok := g.DSDV[dest]
	if ok {
		mutex.Lock()
		g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
		mutex.Unlock()
	} else {
		fmt.Println("destination unknown for data request message")
	}

}