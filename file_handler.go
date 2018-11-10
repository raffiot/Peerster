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
	if fi, err := os.Stat("/._Chunks/"+file_id); !os.IsNotExist(err) {
		buf := make([]byte,int(fi.Size()))
		r, err := os.Open("/._Chunks/"+file_id)
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
	
	mutex.Lock()	
	next_hop, ok := g.DSDV[pkt.Destination]
	if ok {
		pktByte, err := protobuf.Encode(&newPkt)
		if err != nil{
			fmt.Println("Error encoding packet")
			log.Fatal(err)
		}
		g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
		fmt.Println("data request sent")
	} else {
		fmt.Println("destination unknown for data message")
	}
	mutex.Unlock()	
}

func (g *Gossiper) forward_data_msg(pkt *GossipPacket){
	if pkt.DataRequest != nil ||  pkt.DataReply != nil{
		var next_hop string
		var ok bool
		mutex.Lock()
		if pkt.DataRequest != nil{
			next_hop, ok = g.DSDV[pkt.DataRequest.Destination]
		} else{
			next_hop, ok = g.DSDV[pkt.DataReply.Destination]
		}
		
		mutex.Unlock()
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
	newPkt := GossipPacket{DataRequest: &DataRequest{
		Origin: g.Name,
		Destination: pkt.Destination,
		HopLimit: HOP_LIMIT,
		HashValue: []byte(pkt.Request), //Check if correct with copy and everything
	}}
	
	pktByte, err := protobuf.Encode(&newPkt)
	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}
	mutex.Lock()
	next_hop, ok := g.DSDV[pkt.Destination]
	if ok {
		g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
		fmt.Println("data request sent")
	} else {
		fmt.Println("destination unknown for data request message")
	}
	mutex.Unlock()
	
	_,ok = g.file_pending[pkt.Destination]
	f := File{
		Filename: 	pkt.Filename,
		Filesize:	0,
		Metafile:	nil,
		Metahash:	[]byte(pkt.Request),
	}
	if ok {
		g.file_pending[pkt.Destination] = append(g.file_pending[pkt.Destination], f)
	} else {
		var new_array []File
		new_array = append(new_array,f)
		g.file_pending[pkt.Destination] = new_array
	}
}

func (g *Gossiper) receive_file_reply_for_me(pkt *DataReply){
	if pkt.Data != nil {
		
		mutex.Lock()
		pending_array := make([]File, len(g.file_pending[pkt.Origin]))
		copy(pending_array,g.file_pending[pkt.Origin]) 
		mutex.Unlock()
		index := -1
		for i,v := range pending_array {
			if bytes.Equal(pkt.HashValue, v.Metahash){
				//We received first packet for a file
				arr := make([]byte,len(pkt.Data))
				copy(arr,pkt.Data)
				v.Metafile = arr
				downloadPrint(v.Filename,-2, pkt.Origin)
				break
			} else if v.Metafile != nil {
				pkt_waited := v.Filesize / 8192
				//If the packet we receive is confirmed to be the next because the hash is the good one
				if bytes.Equal(pkt.HashValue, v.Metafile[pkt_waited:pkt_waited+32]){
				
					err := ioutil.WriteFile("./._Chunks/"+hex.EncodeToString(pkt.HashValue), pkt.Data, 0644)
					if err != nil {
						fmt.Println("Error during write of hash")
						fmt.Println(err)
						return
					}
					
					f, err := os.OpenFile("./_SharedFiles/"+v.Filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if _, err := f.Write(pkt.Data); err != nil {
						log.Fatal(err)
					}
					if err := f.Close(); err != nil {
						log.Fatal(err)
					}
					
					if pkt_waited+32 >= len(v.Metafile){
						//It was the last packet we waited for this file
						downloadPrint(v.Filename,-1, "")
						index = i
					} else{
						downloadPrint(v.Filename,pkt_waited, pkt.Origin)
						v.Filesize += 8192
						pkt_wanted := v.Filesize / 8192
						newPkt := GossipPacket{DataRequest: &DataRequest{
							Origin: g.Name,
							Destination: pkt.Origin,
							HopLimit: HOP_LIMIT,
							HashValue: v.Metafile[pkt_wanted:pkt_wanted+32],
						}}
						
						pktByte, err := protobuf.Encode(&newPkt)
						if err != nil {
							fmt.Println("Encode of the packet failed")
							log.Fatal(err)
						}
						mutex.Lock()
						next_hop, ok := g.DSDV[pkt.Destination]
						if ok {
							g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
							fmt.Println("data request sent")
						} else {
							fmt.Println("destination unknown for data request message")
						}
						mutex.Unlock()
					}			
				}
				break
			}	
		}
		if index != -1 {
			// File download has been completed
			
			pending_array = append(pending_array[:index],pending_array[index+1:]...)
			//this should modify the reference so its okay
			//g.file_pending[pkt.Origin] = append(g.file_pending[:index],g.file_pending[index+1:]...)
			
		}
		mutex.Lock()
		g.file_pending[pkt.Origin] = pending_array
		mutex.Unlock()
	}

}