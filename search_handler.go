package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"
	"time"

	"github.com/dedis/protobuf"
)

func (g *Gossiper) fwd_search_reply(pkt *SearchReply) {
	next_hop, ok := g.dsdv.state[pkt.Destination]
	if ok {
		hop := pkt.HopLimit - 1
		newPkt := GossipPacket{SearchReply: &SearchReply{
			Origin:      pkt.Origin,
			Destination: pkt.Destination,
			HopLimit:    pkt.HopLimit - 1,
			Results:     pkt.Results,
		}}
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

func (g *Gossiper) checkMatchComplete(matches map[string][]uint64, chunkCount uint64)bool{
	set := make(map[uint64]bool)
	for _, v := range matches  {
		for _, chunk_nb := range v {
			set[chunk_nb] = true
		}
	}
	
	return len(set) == int(chunkCount)
}



func (g *Gossiper) search_reply_for_me(pkt *SearchReply) {
	fmt.Println("search_reply_for_me")
	g.search_matches.m.Lock()
	sm_copy := make([]SearchMatch, len(g.search_matches.sm))
	copy(sm_copy,g.search_matches.sm)
	g.search_matches.m.Unlock()
	for _, searchResultIn := range pkt.Results {
		new_file := true
		matchComplete := false
		//fmt.Println("I'm heere")
		searchPrint(searchResultIn.FileName,pkt.Origin, searchResultIn.MetafileHash, searchResultIn.ChunkMap)
		
		for i, match := range sm_copy {
			if bytes.Equal(match.MetafileHash, searchResultIn.MetafileHash) {
				new_file = false
				g.search_matches.sm[i].Matches[pkt.Origin] = searchResultIn.ChunkMap
				matchComplete = g.checkMatchComplete(g.search_matches.sm[i].Matches,searchResultIn.ChunkCount)
			}
		}
		if new_file {
			//Add new entry to g.search_matches
			matches := make(map[string][]uint64)
			metafileHash := make([]byte, len(searchResultIn.MetafileHash))
			copy(metafileHash, searchResultIn.MetafileHash)
			matches[pkt.Origin] = searchResultIn.ChunkMap
			sm := SearchMatch{
				Filename:     searchResultIn.FileName,
				MetafileHash: metafileHash,
				Matches:      matches,
				ChunkCount:   searchResultIn.ChunkCount,
			}
			g.search_matches.m.Lock()
			g.search_matches.sm = append(g.search_matches.sm, sm)
			g.search_matches.m.Unlock()
			matchComplete = g.checkMatchComplete(sm.Matches,searchResultIn.ChunkCount)
		}
		
		fmt.Println(matchComplete)
		if matchComplete {
			g.pending_search.m.Lock()
			g.pending_search.Nb_match += 1
			if g.pending_search.Nb_match >= 2 {
				//Search finished because we found 2 match
				if g.pending_search.ch != nil {
					g.pending_search.ch <- true
				}
				g.pending_search.Is_pending = false
				g.pending_search.Nb_match = 0
				fmt.Println("SEARCH FINISHED")
			}
			g.pending_search.m.Unlock()
		}
		
	}

	//For each search result in pkt
	//check if entry in search_solutions comparing metafilehash
	//if not download metafile
	//else append chunck map check if the file is complete in case incerement nb match if nb_match ==2 send stop on channel
	//reinitialize pending

}

func (g *Gossiper) receive_search_request(pkt *SearchRequest) {
	fmt.Println("receive search request")
	fmt.Println(pkt)
	var already_received bool

	search_id := pkt.Origin
	for _, elem := range pkt.Keywords {
		search_id += elem
	}
	fmt.Println(search_id)
	bytes:= []byte(search_id)

	sha_256 := sha256.New()
	sha_256.Write(bytes)
	hash := sha_256.Sum(nil)
	search_id = hex.EncodeToString(hash)

	mutex.Lock()
	_, already_received = g.search_req_timeout[search_id]
	mutex.Unlock()

	if !already_received {

		//handle request duplicate
		mutex.Lock()
		g.search_req_timeout[search_id] = true
		mutex.Unlock()
		go g.request_timeout_routine(search_id)

		matching_files := find_matching_files(pkt.Keywords)

		if len(matching_files) > 0 {

			sr := read_files_for_search(matching_files)

			if len(sr) > 0 {
				fmt.Println("found something")
				fmt.Println(sr[0])
				//fmt.Println(sr[1])
				fmt.Println(pkt.Origin)
				newPkt := GossipPacket{SearchReply: &SearchReply{
					Origin:      g.Name,
					Destination: pkt.Origin,
					HopLimit:    HOP_LIMIT,
					Results:     sr,
				}}
				pktByte, err := protobuf.Encode(&newPkt)
				if err != nil {
					log.Fatal(err)
				}
				
				next_hop, ok := g.dsdv.state[pkt.Origin]
				if ok {
					mutex.Lock()
					g.conn.WriteToUDP(pktByte, ParseStrIP(next_hop))
					mutex.Unlock()
				} else {
					fmt.Println("destination unknown for search reply message")
				}
			}
		}
		g.propagate_search(pkt)
	}

}

func (g *Gossiper) propagate_search(pkt *SearchRequest) {
	new_budget := int(pkt.Budget - 1)
	if new_budget > 0 && len(g.set_of_peers.set) > 0 {
		res := new_budget / len(g.set_of_peers.set) //int division so no need of floor
		if res > 0 {
			// Because I don't use lock on set of peers that is append only, I could put more budget than there really is.
			i := new_budget % len(g.set_of_peers.set)
			for dest, _ := range g.set_of_peers.set {
				sending_budget := res
				if i > 0 {
					sending_budget += 1
					i -= 1
				}
				g.send_search(ParseStrIP(dest), sending_budget, pkt)
			}
		} else {
			i := 0
			for ; i < new_budget; i++ {
				dest := g.chooseRandomPeer()
				g.send_search(dest, 1, pkt)
			}
		}
	}
}

func (g *Gossiper) search_routine(pkt *SearchRequest) {
	budget := 2
	finish := false

	for !finish {
		select {
		case _ = <-g.pending_search.ch:
			g.pending_search.ch = nil
			finish = true
		case <-time.After(1 * time.Second):
			fmt.Println("Budget " + string(budget))
			budget *= 2
			pkt.Budget = uint64(budget)
			g.propagate_search(pkt)
			if budget >= 32 {
				finish = true
			}

		}
	}
}

func (g *Gossiper) request_timeout_routine(search_id string) {
	time.Sleep(500 * time.Millisecond)
	mutex.Lock()
	delete(g.search_req_timeout, search_id)
	mutex.Unlock()
}

func (g *Gossiper) send_search(dest *net.UDPAddr, budget int, pkt *SearchRequest) {
	newPkt := GossipPacket{SearchRequest: &SearchRequest{
		Origin:   pkt.Origin,
		Budget:   uint64(budget),
		Keywords: pkt.Keywords,
	}}
	pktByte, err := protobuf.Encode(&newPkt)
	if err != nil {
		log.Fatal(err)
	}
	mutex.Lock()
	g.conn.WriteToUDP(pktByte, dest)
	mutex.Unlock()
}

func find_matching_files(keywords []string) []string {
	//handle file search
	var matching_files []string
	files, err := ioutil.ReadDir("./_SharedFiles") //TO BE COMPLETE WITH ADDING ./_Downloads

	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		for _, k := range keywords {
			re := regexp.MustCompile(".*" + k + ".*")
			if re.MatchString(f.Name()) {
				fmt.Println(f.Name())
				ok := contains(matching_files, f.Name())

				if !ok {
					matching_files = append(matching_files, f.Name())
				}
			}
		}
	}
	fmt.Println(matching_files)
	return matching_files
}

func read_files_for_search(matching_files []string) []*SearchResult {
	var sr []*SearchResult

	for _, k := range matching_files {
		fmt.Println(k)
		var chunks []uint64
		var metafile []byte

		r, err := os.Open("./_SharedFiles/" + k)
		if err != nil {
			fmt.Printf("error opening file: %v\n", err)
			os.Exit(1)
		}

		fi, e := os.Stat("./_SharedFiles/" + k)
		if e != nil {
			log.Fatal(e)
			os.Exit(1)
		}

		filesize := int(fi.Size())

		file_size_rem := filesize

		var chunck_counter uint64
		chunck_counter = 1

		//repeat read as long as there is data
		for file_size_rem > 0 {
			buf_size := 8192
			if file_size_rem < 8192 {
				buf_size = file_size_rem
			}
			buf := make([]byte, buf_size)

			_, err := r.Read(buf)

			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}

			sha_256 := sha256.New()
			sha_256.Write(buf)
			hash := sha_256.Sum(nil)
			metafile = append(metafile, hash...)
			file_size_rem -= 8192
			//I reply the file only if it has been upload before
			if _, err := os.Stat("./._Chunks/" + hex.EncodeToString(hash)); !os.IsNotExist(err) {
				chunks = append(chunks, chunck_counter)
			}

			chunck_counter += 1
		}
		if len(chunks) > 0 {
			sha_256 := sha256.New()
			sha_256.Write(metafile)
			metafileHash := sha_256.Sum(nil)
			chcount := uint64(len(metafile)/32)
			sr_file := SearchResult{
				FileName:     k,
				MetafileHash: metafileHash,
				ChunkMap:     chunks,
				ChunkCount:	  chcount,
			}

			sr = append(sr, &sr_file)
		}
	}
	return sr
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
