package main

import (
	"fmt"
	"os"
	"io"
	"log"
	"crypto/sha256"
	"encoding/hex"
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
		metafile = append(metafile,sha_256.Sum(nil)...)
		file_size_rem -= 8192
		
	}
	sha_256 := sha256.New()
	sha_256.Write(metafile)
	metahash := sha_256.Sum(nil)
	
	
	file := File{
		Filename: filename,
		Filesize: filesize,
		Metafile: metafile,
		Metahash: metahash,
	}
	fmt.Println(filename)
	fmt.Println(filesize)
	fmt.Println(hex.EncodeToString(metafile))
	fmt.Println(hex.EncodeToString(metahash))
	g.files[string(metahash)] = file
	return
}

func (g *Gossiper) send_file_request(pkt *FileMessage){

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
}

func (g *Gossiper) receive_file_request_for_me(pkt *DataRequest){
	
	// FIRST REQUEST !!! WE HAVE TO KEEP A STATE !!!!
	mutex.Lock()
	file, ok := g.files[string(pkt.HashValue)]
	mutex.Unlock()
	if ok{
		mutex.Lock()
		newPkt := GossipPacket{DataReply: &DataReply{
			Origin: g.Name,
			Destination: pkt.Origin,
			HopLimit: HOP_LIMIT,
			HashValue: []byte(file.Metahash), //Check if correct with copy and everything
			Data: file.Metafile,
		}}
		
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
	} else {
		fmt.Println("I don't have this file")
	}	
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