package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"github.com/dedis/protobuf"
	"time"
)

func (g *Gossiper) tx_receive(pkt *TxPublish){
	fmt.Println("tx_receive "+pkt.File.Name)
	already_seen := false
	g.blockchain.m.Lock()
	for _,tx := range g.blockchain.Pending{
		if tx.File.Name == pkt.File.Name {
			already_seen = true
			break
		}
	}
	_,ok := g.blockchain.FileMapping[pkt.File.Name]
	if !already_seen && !ok {
		g.blockchain.Pending = append(g.blockchain.Pending,*pkt)
	}
	g.blockchain.m.Unlock()
	
	if !already_seen && !ok {
		
		newPkt := &GossipPacket{TxPublish: &TxPublish{
			File: pkt.File,
			HopLimit: pkt.HopLimit - 1,
		}}
		
		if newPkt.TxPublish.HopLimit > 0 {
			
			g.broadcastBlockchain(newPkt)
		}
	}
	
	//Check !receive in PendingTx
	//Check Tx isn't in blockchain (filename)
	//Test true true:
		//yes: store in PendingTx, decrement HOP_LIMIT and broadcast
		//no: discard
		
	
}

func (g *Gossiper) block_receive(pkt *BlockPublish){
	
	//Check if prev_hash in all_b or if prev_hash = 0 --> valid
	//Check if hash_block_rcv in all_b --> already_seen
	//Check if PoW correct --> valid

	empty_test_2 := make([]byte,2)
	//empty_test_32 := make([]byte,32)
	
	hash_block_rcv := pkt.Block.Hash()
	hash_prev_rcv := pkt.Block.PrevHash
	hash_block_rcv_str := hex.EncodeToString(hash_block_rcv[:]) 
	hash_prev_rcv_str := hex.EncodeToString(hash_prev_rcv[:]) 
	
	g.blockchain.m.RLock()
	_,have_prev := g.blockchain.All_b[hash_prev_rcv_str]	
	_,already_seen := g.blockchain.All_b[hash_block_rcv_str]
	g.blockchain.m.RUnlock()
	
	
	//have_prev = have_prev || (bytes.Equal(hash_prev_rcv[:],empty_test_32) && len(g.blockchain.Longest) ==0)
	have_prev = have_prev || len(g.blockchain.Longest) ==0
	valid_block := bytes.Equal(hash_block_rcv[:2],empty_test_2)
	
	if !already_seen && valid_block && have_prev{
		
		g.update_blockchain(pkt.Block)
		
		newpkt := &GossipPacket{BlockPublish: &BlockPublish{
			Block: pkt.Block,
			HopLimit: pkt.HopLimit - 1,
		}}
		if newpkt.BlockPublish.HopLimit > 0 {
			g.broadcastBlockchain(newpkt)
		}
	} else if !already_seen && valid_block {
		//HAPPEN TO LONELY BLOCK 
		//because block has no prev found
		g.lonely_blocks.m.Lock()
		g.lonely_blocks.lonelys[hash_prev_rcv_str] = &(pkt.Block)
		g.lonely_blocks.m.Unlock()
				
	}
	
	//Check HashPrev in == block hash in blockchain (if blockchain empty, accept)
	//check PoW correct (hashblock and check 16 0bits)
	//Check !already seen
	//test true:
		//yes: add block to blockchain, add block to mapping, broadcast with decrement HOP_LIMIT
		//withdraw tx of block that also are in pendingtx
		
}


func (g *Gossiper) mine(){
	first := true
	for true{
		
		g.blockchain.m.RLock()
		pendings := make([]TxPublish,len(g.blockchain.Pending))
		copy(pendings,g.blockchain.Pending) // Does it overconsume memory?? Or it is free at the end?
		g.blockchain.m.RUnlock()
		
		g.blockchain.m.RLock()
		var prevhash [32]byte
		if len(g.blockchain.Longest) > 0 {
			prevhash = g.blockchain.Longest[len(g.blockchain.Longest)-1].bl.Hash()
		}
		g.blockchain.m.RUnlock()
		
		var nonce [32]byte
		rand.Read(nonce[:])
		
		
		bl := Block{
			PrevHash: prevhash,
			Nonce: nonce,
			Transactions: pendings,
		}
		
		resHash := bl.Hash()
		empty_test := make([]byte,2)
		if bytes.Equal(resHash[:2],empty_test) {
			
			if first {
				time.Sleep(time.Duration(SLEEP_FIRST_BLOCK) * time.Second)
				first = false
			}
			
			printFoundBlock(hex.EncodeToString(resHash[:]))
			fmt.Println(hex.EncodeToString(prevhash[:]))
			g.update_blockchain(bl)
			
			
			newpkt := &GossipPacket{ BlockPublish: &BlockPublish{
					Block:	bl,
					HopLimit: HOP_LIMIT_BLOCK,
			}}
			
			g.broadcastBlockchain(newpkt)
				
		}
		
	}
	//While true
		//Read PendingTx
		//Random Nounce
		//Create Block
		//Hash
		//Test if 16 0bits
			//yes: add block to blockchain, broadcast, delete pendingtx, update FileMapping
			//no: repeat
}

func (g * Gossiper) update_blockchain(bl Block){
	
	hash_bl:= bl.Hash()
	hash_bl_str := hex.EncodeToString(hash_bl[:]) 
	hash_prev_str := hex.EncodeToString(bl.PrevHash[:])

	var longest *BlockWithLink
	g.blockchain.m.RLock()
	length := len(g.blockchain.Longest)
	g.blockchain.m.RUnlock()
	if length > 0 { 
		//If it isnt our first block
		g.blockchain.m.RLock()
		longest = g.blockchain.Longest[len(g.blockchain.Longest) - 1]
		g.blockchain.m.RUnlock()
		longest_hash := longest.bl.Hash()
		
		if bytes.Equal(bl.PrevHash[:],longest_hash[:]){	
			//The block extend our longest chain
			lgn := g.extendChain(g.blockchain.Longest,bl)
			lgn = g.happen_lonelys(lgn)
			//lgn_tx := get_transactions(lgn)
			//CHECK IF I CAN HAPPEN A LONELY BLOCK
			g.blockchain.m.Lock()
			//VERIFIY DUPLICATE BEFORE UPDATE
			//block_ok := g.blockNoDuplicate(lgn_tx)
			block_ok := g.blockNoDuplicate(bl.Transactions) //I have to also check lonely if it have been append
			if block_ok {
				g.blockchain.Longest = lgn
				g.updatePendings(bl.Transactions)
				g.updateFileMapping(bl.Transactions)
			}
			g.blockchain.m.Unlock()	
					
			
			fmt.Println("extend longest")
			if block_ok {
				printChain(g.blockchain.Longest[len(g.blockchain.Longest)-1])
			}
		} else{
			g.blockchain.m.RLock()
			tab,ok := g.blockchain.All_c[hash_prev_str] //Does it return real chain pointer
			g.blockchain.m.RUnlock()
			if ok { //There exist a chain that we extend
			
				tb := g.extendChain(tab,bl)
				tb = g.happen_lonelys(tb)
				fmt.Println("extend existing chain")
				//CHECK IF I CAN HAPPEN A LONELY BLOCK
				if len(tb) > len(g.blockchain.Longest) {
					g.rollBack(tb)
					
				} else {
					common_ancestor,_ := findCommonAncestor(g.blockchain.Longest[len(g.blockchain.Longest)-1],tb)
					printForkShorter(common_ancestor.Hash())
				}
			} else { //We create our own chain
				g.blockchain.m.RLock()
				prev, ok := g.blockchain.All_b[hash_prev_str]
				g.blockchain.m.RUnlock()
				if ok { //if we have a previous block
					

					fmt.Println("extend new chain")

					new_chain := buildChainFromBlock(prev)
					new_chain = g.happen_lonelys(new_chain)
					//CHECK IF I CAN HAPPEN A LONELY BLOCK
					
					new_bl := &BlockWithLink{
						bl:bl,
						prev: prev,
					}

					prev.next = append(prev.next,new_bl)
					new_chain = append(new_chain,new_bl)

					g.blockchain.m.Lock()
					g.blockchain.All_b[hash_bl_str] = new_bl
					g.blockchain.All_c[hash_bl_str] = new_chain
					g.blockchain.m.Unlock()
					if len(new_chain) > len(g.blockchain.Longest) {
						g.rollBack(new_chain)
					} else {
						common_ancestor,_ := findCommonAncestor(g.blockchain.Longest[len(g.blockchain.Longest)-1],new_chain)
						printForkShorter(common_ancestor.Hash())
					}
					
				}
				
			}
		}
	} else {
		fmt.Println("create new longest")
		var new_chain []*BlockWithLink
		new_bl_ln := &BlockWithLink{bl:bl}
		new_chain = append(new_chain,new_bl_ln)
		g.blockchain.m.Lock()
		// ITS FIRST CHAIN SO NO CHECK FOR DUPLICATE
		g.blockchain.All_b[hash_bl_str] = new_bl_ln
		g.blockchain.All_c[hash_bl_str] = new_chain
		g.blockchain.Longest = new_chain
		g.updatePendings(bl.Transactions)
		g.updateFileMapping(bl.Transactions)	
		g.blockchain.m.Unlock()
		
	}
	
}

func get_transactions(chain []*BlockWithLink) ([]TxPublish){
	var array []TxPublish
	for _,c := range chain{
		for _,tx_pub := range c.bl.Transactions{
			array = append(array,tx_pub)
		}
	}
	return array
}

func (g *Gossiper) blockNoDuplicate(tx []TxPublish) (bool){
	no_duplicate := true
	for _,elem := range tx {
		_,ok := g.blockchain.FileMapping[elem.File.Name]
		no_duplicate = no_duplicate && !ok 
	}
	return no_duplicate
}

func (g *Gossiper) happen_lonelys(chain []*BlockWithLink)([]*BlockWithLink){
	latest := chain[len(chain)-1]
	latest_hash := latest.bl.Hash()
	latest_hash_str := hex.EncodeToString(latest_hash[:])
	g.lonely_blocks.m.Lock()
	elem,ok := g.lonely_blocks.lonelys[latest_hash_str]
	if ok {
		var next_array []*BlockWithLink
		blockL := &BlockWithLink{
				bl: *elem,
				prev: latest,
		}
		next_array = append(next_array,blockL)
		latest.next = next_array
		chain = append(chain,blockL)
		delete(g.lonely_blocks.lonelys,latest_hash_str)
	}
	g.lonely_blocks.m.Unlock()
	return chain
}

func compare_tx(old_longest []*BlockWithLink, new_longest []*BlockWithLink) (ensemble_a []TxPublish, ensemble_b []TxPublish){

	var old_tx []TxPublish
	var new_tx []TxPublish

	for _,bl1 := range old_longest{
		old_tx = append(old_tx,bl1.bl.Transactions...)
	}
	for _,bl2 := range new_longest{
		new_tx = append(new_tx,bl2.bl.Transactions...)
	}

	for _,tx1 := range old_tx {
		not_found := true
		for _,tx2 := range new_tx {
			if tx1.File.Name == tx2.File.Name {
				not_found = false
			}
		}
		if not_found {
			ensemble_a = append(ensemble_a,tx1)
		}
	}

	for _,tx2 := range new_tx {
		not_found := true
		for _,tx1 := range old_tx {
			if tx1.File.Name == tx2.File.Name {
				not_found = false
			}
		}
		if not_found {
			ensemble_b = append(ensemble_a,tx2)
		}
	}
	return
}

func (g *Gossiper) rollBack(new_longest []*BlockWithLink){
	//printChain(g.blockchain.Longest[len(g.blockchain.Longest)-1])

	g.blockchain.m.RLock()
	longest_node := g.blockchain.Longest[len(g.blockchain.Longest) - 1]
	g.blockchain.m.RUnlock()
	
	common_ancestor,counter := findCommonAncestor(longest_node,new_longest)
	if common_ancestor == nil {
		fmt.Println("Error, no common ancestor")
		return
	}
	printForkLonger(counter)

	//VERIFIY DUPLICATE BEFORE UPDATE
	

	g.blockchain.m.Lock()
	ensemble_a, ensemble_b := compare_tx(g.blockchain.Longest,new_longest)
	chain_ok := g.blockNoDuplicate(ensemble_b)
	if chain_ok {
		g.blockchain.Longest = new_longest

		for _,tx := range ensemble_a {
			delete(g.blockchain.FileMapping,tx.File.Name)	
			g.blockchain.Pending = append(g.blockchain.Pending, tx) 		
		}
		for _,tx := range ensemble_b {
			g.blockchain.FileMapping[tx.File.Name] = tx.File.MetafileHash
		}
	}
	g.blockchain.m.Unlock()
	if chain_ok {
		printChain(g.blockchain.Longest[len(g.blockchain.Longest)-1])
	}

	//Find common ancestor
	//Undo FileMapping until common ancestor
	//Go up on chain and sum up Transactions
	//call updatePendings with this list
	//call update_fileMapping with this list
}

func findCommonAncestor(longest_node *BlockWithLink,new_longest []*BlockWithLink) (*Block,int){
	elem_undo := longest_node
	counter := 1
	for elem_undo.prev != nil {
		elem_undo = elem_undo.prev
		elem_undo_hash := elem_undo.bl.Hash()
		for _,elem_redo := range new_longest{
			elem_redo_hash := elem_redo.bl.Hash()
			if bytes.Equal(elem_undo_hash[:],elem_redo_hash[:]){
				return &(elem_undo.bl),counter
			}
		}
		counter +=1
	}
	return nil,counter
}

func (g *Gossiper) extendChain(tab []*BlockWithLink, block Block) ([]*BlockWithLink){

				
	prev := tab[len(tab)-1]
	prev_hash := prev.bl.Hash()
	block_hash := block.Hash()
	hash_bl_str := hex.EncodeToString(block_hash[:])
	hash_prev_str := hex.EncodeToString(prev_hash[:])
	blockL := &BlockWithLink{
		bl: block,
		prev: prev,
	}
	var next_array []*BlockWithLink
	next_array = append(next_array,blockL)
	prev.next = next_array
	
	
	
	g.blockchain.m.Lock()
	
	tab = append(tab, blockL)
	fmt.Println("Here")
	g.blockchain.All_b[hash_bl_str] = blockL
	_,ok := g.blockchain.All_c[hash_prev_str]
	if ok{
		delete(g.blockchain.All_c,hash_prev_str)
		g.blockchain.All_c[hash_bl_str] = tab
	}
	g.blockchain.m.Unlock()
	return tab 
}

func buildChainFromBlock(block *BlockWithLink) ([]*BlockWithLink) {
	var new_chain []*BlockWithLink
	var first = block

	for first != nil{
		new_chain = append([]*BlockWithLink{first},new_chain...)
		first = first.prev
	}

	//THE CHAIN DOESN'T CONTAIN block BECAUSE WE WANT A COPY of it
	
	return new_chain
}

func (g *Gossiper) updatePendings(pendings []TxPublish){
	var new_pendings []TxPublish
	for _,tx1 := range g.blockchain.Pending {
		found := false
			for _, tx2 := range pendings {
				if tx1.File.Name == tx2.File.Name{
					found = true
				}
			}
		if !found {
			//fmt.Println("NOT FOUND")
			new_pendings = append(new_pendings,tx1)
		}
	}
	g.blockchain.Pending = new_pendings
}

func (g *Gossiper) updateFileMapping(pendings []TxPublish){
	for _, tx := range pendings{
		g.blockchain.FileMapping[tx.File.Name] = tx.File.MetafileHash
	}
}

func (g* Gossiper) newFileNotice(name string, size int, metafilehash []byte){

	b := make([]byte,len(metafilehash))
	copy(b,metafilehash)
	
	f := File{
		Name:	name,
		Size:	int64(size),
		MetafileHash:	b,
	}
	
	txpb := TxPublish{
		File:	f,
		HopLimit:	HOP_LIMIT_TX,
	}
	
	
	already_seen := false
	g.blockchain.m.Lock()
	for _,tx := range g.blockchain.Pending{
		if tx.File.Name == txpb.File.Name {
			already_seen = true
			break
		}
	}
	if !already_seen {
		g.blockchain.Pending = append(g.blockchain.Pending, txpb)
	}
	g.blockchain.m.Unlock()
	
	gp := &GossipPacket{TxPublish : &txpb}
	g.broadcastBlockchain(gp)
}

func (b *Block) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	h.Write(b.Nonce[:])
	binary.Write(h,binary.LittleEndian,uint32(len(b.Transactions)))
	for _, t := range b.Transactions {
		th := t.Hash()
		h.Write(th[:])
	}
	copy(out[:], h.Sum(nil))
	return
}

func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h,binary.LittleEndian,uint32(len(t.File.Name)))
	h.Write([]byte(t.File.Name))
	h.Write(t.File.MetafileHash)
	copy(out[:], h.Sum(nil))
	return
}

func (g *Gossiper) broadcastBlockchain(newPkt *GossipPacket){
	
	packet_encoded, err := protobuf.Encode(newPkt)
	if err != nil {
		fmt.Println("cannot encode message")
		log.Fatal(err)
	}
	
	for k := range g.set_of_peers.set {
		dst, err := net.ResolveUDPAddr("udp4", k)
		if err != nil {
			fmt.Println("cannot resolve addr of other gossiper")
			log.Fatal(err)
		}
		mutex.Lock()
		g.conn.WriteToUDP(packet_encoded, dst)
		mutex.Unlock()
	}
}
