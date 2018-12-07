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
)

func (g *Gossiper) tx_receive(pkt *TxPublish){
	
	already_seen := false
	g.pending_tx.m.RLock()
	for _,tx := range g.pending_tx.Pending{
		if tx.File.Name == pkt.File.Name {
			already_seen = true
			break
		}
	}
	g.pending_tx.m.RUnlock()
	
	g.file_mapping.m.RLock()
	_,ok := g.file_mapping.FileMapping[pkt.File.Name]
	g.file_mapping.m.RUnlock()
	
	if !already_seen && !ok {
		g.pending_tx.m.Lock()
		g.pending_tx.Pending = append(g.pending_tx.Pending,*pkt)
		g.pending_tx.m.Unlock()
		
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
	empty_test_32 := make([]byte,32)
	
	hash_block_rcv := pkt.Block.Hash()
	hash_prev_rcv := pkt.Block.PrevHash
	hash_block_rcv_str := hex.EncodeToString(hash_block_rcv[:]) 
	hash_prev_rcv_str := hex.EncodeToString(hash_prev_rcv[:]) 
	
	g.blockchain.m.RLock()
	_,have_prev := g.blockchain.All_b[hash_prev_rcv_str]	
	_,already_seen := g.blockchain.All_b[hash_block_rcv_str]
	g.blockchain.m.RUnlock()
	
	
	have_prev = have_prev || (bytes.Equal(hash_prev_rcv[:],empty_test_32) && len(g.blockchain.Longest) ==0)
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
	}
	
	//Check HashPrev in == block hash in blockchain (if blockchain empty, accept)
	//check PoW correct (hashblock and check 16 0bits)
	//Check !already seen
	//test true:
		//yes: add block to blockchain, add block to mapping, broadcast with decrement HOP_LIMIT
		//withdraw tx of block that also are in pendingtx
		
}


func (g *Gossiper) mine(){
	for true{
		
		g.pending_tx.m.RLock()
		pendings := make([]TxPublish,len(g.pending_tx.Pending))
		copy(pendings,g.pending_tx.Pending) // Does it overconsume memory?? Or it is free at the end?
		g.pending_tx.m.RUnlock()
		
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
		
			printFoundBlock(hex.EncodeToString(resHash[:]))
			
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
			g.extendChain(g.blockchain.Longest,bl)		
			g.updatePendings(bl.Transactions)		
			g.updateFileMapping(bl.Transactions)
			printChain(g.blockchain.Longest[len(g.blockchain.Longest)-1])
		} else{
			g.blockchain.m.RLock()
			tab,ok := g.blockchain.All_c[hash_prev_str] //Does it return real chain pointer
			g.blockchain.m.RUnlock()
			if ok { //There exist a chain that we extend
			
				g.extendChain(tab,bl)			
				
				if len(tab) > len(g.blockchain.Longest) {
					g.rollBack(tab)
					printChain(g.blockchain.Longest[len(g.blockchain.Longest)-1])
				} else {
					common_ancestor,_ := findCommonAncestor(g.blockchain.Longest[len(g.blockchain.Longest)-1],tab)
					printForkShorter(common_ancestor.Hash())
				}
			} else { //We create our own chain
				g.blockchain.m.RLock()
				prev, ok := g.blockchain.All_b[hash_prev_str]
				g.blockchain.m.RUnlock()
				if ok { //if we have a previous block
				
					new_chain := buildChainFromBlock(prev)
		
					prev_copy := &BlockWithLink{
						bl: prev.bl,
						prev: prev.prev,
					}
					new_chain = append(new_chain,prev_copy)
					
					new_bl := &BlockWithLink{
						bl:bl,
						prev: prev_copy,
					}
					prev_copy.next = new_bl
					new_chain = append(new_chain,new_bl)
					g.blockchain.m.Lock()
					g.blockchain.All_b[hash_bl_str] = new_bl
					g.blockchain.All_c[hash_bl_str] = new_chain
					g.blockchain.m.Unlock()
					if len(new_chain) > len(g.blockchain.Longest) {
						g.rollBack(new_chain)
						printChain(g.blockchain.Longest[len(g.blockchain.Longest)-1])
					} else {
						common_ancestor,_ := findCommonAncestor(g.blockchain.Longest[len(g.blockchain.Longest)-1],new_chain)
						printForkShorter(common_ancestor.Hash())
					}
					
				}
			}
		}
	} else {
		var new_chain []*BlockWithLink
		new_bl_ln := &BlockWithLink{bl:bl}
		new_chain = append(new_chain,new_bl_ln)
		g.blockchain.m.Lock()
		g.blockchain.All_b[hash_bl_str] = new_bl_ln
		g.blockchain.All_c[hash_bl_str] = new_chain
		g.blockchain.Longest = new_chain
		g.blockchain.m.Unlock()
		
	}
	
}

func (g *Gossiper) rollBack(new_longest []*BlockWithLink){
	
	g.blockchain.m.RLock()
	longest_node := g.blockchain.Longest[len(g.blockchain.Longest) - 1]
	g.blockchain.m.RUnlock()
	
	common_ancestor,counter := findCommonAncestor(longest_node,new_longest)
	if common_ancestor == nil {
		fmt.Println("Error, no common ancestor")
		return
	}
	printForkLonkger(counter)
	
	g.rollback_fileMapping(longest_node,new_longest[len(new_longest)-1] ,common_ancestor)

	//Find common ancestor
	//Undo FileMapping until common ancestor
	//Go up on chain and sum up Transactions
	//call updatePendings with this list
	//call update_fileMapping with this list
}

func (g *Gossiper) rollback_fileMapping(old_chain *BlockWithLink, new_chain *BlockWithLink, common_ancestor *Block){
	

	current_node := old_chain
	current_hash := old_chain.bl.Hash()
	ancestor_hash := common_ancestor.Hash()
	finish := false
	g.file_mapping.m.Lock()
	for !bytes.Equal(current_hash[:],ancestor_hash[:]) && !finish {
		for _,tx := range current_node.bl.Transactions{
			delete(g.file_mapping.FileMapping,tx.File.Name)
		}
		if current_node.prev != nil {
			current_node = current_node.prev
		} else {
			finish = true
		}
	}
	g.file_mapping.m.Unlock()
	
	current_node = new_chain
	current_hash = new_chain.bl.Hash()
	finish = false
	var pendings []TxPublish
	
	g.file_mapping.m.Lock()
	for !bytes.Equal(current_hash[:],ancestor_hash[:]) && !finish {
		for _,tx := range current_node.bl.Transactions{
			g.file_mapping.FileMapping[tx.File.Name] = tx.File.MetafileHash
			pendings = append(pendings,tx)
		}
		if current_node.prev != nil {
			current_node = current_node.prev
		} else {
			finish = true
		}
	}
	g.file_mapping.m.Unlock()
	g.updatePendings(pendings)
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

func (g *Gossiper) extendChain(tab []*BlockWithLink, block Block){

				
	prev := tab[len(tab)-1]
	prev_hash := prev.bl.Hash()
	block_hash := block.Hash()
	hash_bl_str := hex.EncodeToString(block_hash[:])
	hash_prev_str := hex.EncodeToString(prev_hash[:])
	blockL := &BlockWithLink{
		bl: block,
		prev: prev,
	}
	
	prev.next = blockL
	
	
	
	g.blockchain.m.Lock()
	
	tab = append(tab, blockL)
	g.blockchain.All_b[hash_bl_str] = blockL
	_,ok := g.blockchain.All_c[hash_prev_str]
	if ok{
		delete(g.blockchain.All_c,hash_prev_str)
		g.blockchain.All_c[hash_bl_str] = tab
	}
	g.blockchain.m.Unlock()
}

func buildChainFromBlock(block *BlockWithLink) ([]*BlockWithLink) {
	block_hash := block.bl.Hash()
	var first = block

	for first.prev != nil{
		first = first.prev
	}
	

	var new_chain []*BlockWithLink
	var first_hash = first.bl.Hash()
	for !bytes.Equal(first_hash[:],block_hash[:]){
		new_chain = append(new_chain, first)
		first = first.next
		first_hash = first.bl.Hash()
	}

	//THE CHAIN DOESN'T CONTAIN block BECAUSE WE WANT A COPY of it
	
	return new_chain
}

func (g *Gossiper) updatePendings(pendings []TxPublish){
	g.pending_tx.m.Lock()
	var new_pendings []TxPublish
	for _,tx1 := range g.pending_tx.Pending {
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
	g.pending_tx.Pending = new_pendings
	g.pending_tx.m.Unlock()
}

func (g *Gossiper) updateFileMapping(pendings []TxPublish){
	g.file_mapping.m.Lock()
	for _, tx := range pendings{
		g.file_mapping.FileMapping[tx.File.Name] = tx.File.MetafileHash
	}
	g.file_mapping.m.Unlock()
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
	g.pending_tx.m.Lock()
	for _,tx := range g.pending_tx.Pending{
		if tx.File.Name == txpb.File.Name {
			already_seen = true
			break
		}
	}
	if !already_seen {
		g.pending_tx.Pending = append(g.pending_tx.Pending, txpb)
	}
	g.pending_tx.m.Unlock()
	
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
