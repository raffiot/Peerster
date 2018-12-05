package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"github.com/dedis/protobuf"
)

func (g *Gossiper) tx_receive(pkt *TxPublish){
	
	already_seen := false
	g.pending_tx.m.Lock()
	for _,tx := range g.pending_tx.Pending{
		if tx.File.Name == pkt.File.Name {
			already_seen = true
			break
		}
	}
	g.pending_tx.m.Unlock()
	
	g.file_mapping.m.Lock()
	_,ok := g.file_mapping.FileMapping[pkt.File.Name]
	g.file_mapping.m.Unlock()
	
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

	valid_block := false
	already_seen := false
	hash_block_rcv := pkt.Block.Hash()
	g.blockchain.m.Lock()
	if len(g.blockchain.Blockchain) == 0 {
		valid_block = true
	} else {
		for _,block := range g.blockchain.Blockchain{
			hash_block := block.Hash()
			if bytes.Equal(hash_block[:],pkt.Block.PrevHash[:]){
				valid_block = true
			}
			if bytes.Equal(hash_block[:],hash_block_rcv[:]){
				already_seen = true
			}	
		}
	}
	
	valid_block = valid_block && (hash_block_rcv[0] == 0) && (hash_block_rcv[1] == 0)
	
	if !already_seen && valid_block{

		transaction_to_import := make([]TxPublish,len(pkt.Block.Transactions))
		copy(transaction_to_import,pkt.Block.Transactions)
		
		g.update_blockchain(pkt.Block, transaction_to_import)
		
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
		
		g.pending_tx.m.Lock()
		pendings := make([]TxPublish,len(g.pending_tx.Pending))
		copy(pendings,g.pending_tx.Pending) // Does it overconsume memory?? Or it is free at the end?
		g.pending_tx.m.Unlock()
		g.blockchain.m.Lock()
		var prevhash [32]byte
		if len(g.blockchain.Blockchain) > 0 {
			prevhash = g.blockchain.Blockchain[len(g.blockchain.Blockchain)-1].Hash()
		}
		g.blockchain.m.Unlock()
		
		var nonce [32]byte
		rand.Read(nonce[:])
		
		
		bl := Block{
			PrevHash: prevhash,
			Nonce: nonce,
			Transactions: pendings,
		}
		
		resHash := bl.Hash()
		
		if resHash[0] == 0 && resHash[1] == 0 {
		
			g.update_blockchain(bl,pendings)
			
			
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

func (g * Gossiper) update_blockchain(bl Block, pendings []TxPublish){
	
	//No need to deep copy block as it will not be modified
	//bl_copy := copyBlock(bl)
	g.blockchain.m.Lock()
	g.blockchain.Blockchain = append(g.blockchain.Blockchain, bl)
	g.blockchain.m.Unlock()
			
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
			new_pendings = append(new_pendings,tx1)
		}
	}
	g.pending_tx.Pending = new_pendings
	g.pending_tx.m.Unlock()
			
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