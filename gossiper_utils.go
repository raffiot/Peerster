package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"encoding/hex"
	"sync"
)

type GossipPacket struct {
	Simple        *SimpleMessage
	Rumor         *RumorMessage
	Status        *StatusPacket
	Private       *PrivateMessage
	DataRequest   *DataRequest
	DataReply     *DataReply
	SearchRequest *SearchRequest
	SearchReply   *SearchReply
	TxPublish 	  *TxPublish
	BlockPublish  *BlockPublish
}

type ClientPacket struct {
	Simple  *SimpleMessage
	Private *PrivateMessage
	File    *FileMessage
	Search  *SearchRequest
}

/**
udp_address: Gossiper udp address
conn: Gossiper-other gossiper udp connexion
Name: Identifier of the gossiper
set_of_peers: set of neighbouring peers
vector_clock: vector clock of my rumors already received
archives: array of messages received organized by peers identifier
simple: is simple mode activated
clientConn: Gossiper-client udp connexion
clientAddr: Client udp address


*/
type Gossiper struct {
	udp_address *net.UDPAddr
	conn        *net.UDPConn

	Name               string
	set_of_peers       Set_of_peers
	rumor_state	   Rumor_state
	archives_private   Private_state
	dsdv               DSDV
	simple             bool
	clientConn         *net.UDPConn
	clientAddr         *net.UDPAddr
	rtimer             int
	file_pending       PendingFiles
	rumor_acks         RumorAcks
	search_req_timeout map[string]bool
	search_matches     SearchMatches
	pending_search     PendingSearches
	blockchain		   Blockchain
	lonely_blocks	   LonelyBlocks
}

type Rumor_state struct {
	vector_clock       []PeerStatus
	archives           []PeerMessage
	m		   sync.RWMutex
}

type Private_state struct {
	archives           map[string][]PrivateMessage
	m		   sync.Mutex
}

type Set_of_peers struct {
	set	map[string]bool
	m	sync.Mutex
}

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

type FileMessage struct {
	Destination string
	Filename    string
	Request     string
}

type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

type PeerStatus struct {
	Identifier string
	NextID     uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

type PeerMessage struct {
	Identifier string
	msgs       map[uint32]*RumorMessage
}

type FileForSearch struct {
	Filename string
	Filesize int
	Metafile []byte
	Metahash []byte
	FromDests []string
	ack	 AckFile
}

type AckFile struct {
	hashExpected	string
	dest	string
	ch		chan bool
}

type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
}

type PendingFiles struct{
	pf	map[string]*FileForSearch
	m 	sync.Mutex
}

type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}

type RumorAcks struct {
	racks	map[string][]AckRumor
	m	sync.Mutex
}

type AckRumor struct {
	Identifier string
	NextID     uint32
	ch         chan bool
}



type SearchRequest struct {
	Origin   string
	Budget   uint64
	Keywords []string
}

type SearchReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	Results     []*SearchResult
}

type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount	 uint64
}

type SearchMatch struct {
	Filename     string
	MetafileHash []byte
	Matches      map[string][]uint64
	ChunkCount   uint64
}

type SearchMatches struct {
	sm		[]SearchMatch
	m		sync.RWMutex
}

type PendingSearches struct {
	ps	[]*PendingSearch
	m	sync.Mutex
}

type PendingSearch struct {
	Is_pending bool
	Nb_match   int
	ch         chan bool
	keywords   []string
}



type DSDV struct{
	state	map[string]string
	m	sync.Mutex
}

type TxPublish struct {
	File File
	HopLimit uint32
}

type BlockPublish struct {
	Block Block
	HopLimit uint32
}

type File struct {
	Name string
	Size int64
	MetafileHash []byte
}

type Block struct {
	PrevHash [32]byte
	Nonce [32]byte
	Transactions []TxPublish
}

/**
type Blockchain struct {
	Blockchain []Block
	m		   sync.RWMutex
}*/
type Blockchain struct{
	Longest	[]*BlockWithLink
	All_c	map[string][]*BlockWithLink
	All_b	map[string]*BlockWithLink
	already_seen	map[string]bool
	FileMapping	map[string][]byte
	Pending		[]TxPublish
	m		sync.RWMutex
}



type BlockWithLink struct{
	bl Block
	prev	*BlockWithLink
	next	[]*BlockWithLink
}

type LonelyBlocks struct {
	lonelys			map[string]*Block
	m				sync.RWMutex
}

/**
Constructor of Gossiper
*/
func NewGossiper(address string, name string, peers []string, simple bool, client string, timer int) *Gossiper {
	udpAddr, err := net.ResolveUDPAddr("udp4", address)
	udpConn, err := net.ListenUDP("udp4", udpAddr)

	if err != nil {
		fmt.Println("Error creating new gossiper !")
		log.Fatal(err)
	}
	udpAddrClient, err := net.ResolveUDPAddr("udp4", client)
	udpConnClient, err := net.ListenUDP("udp4", udpAddrClient)

	if err != nil {
		fmt.Println("Error creating new gossiper !")
		log.Fatal(err)
	}


	var sm []SearchMatch
	var mutex = sync.RWMutex{}
	var search_matches = SearchMatches{
		sm: sm,
		m: mutex,
	}
	
	
	var ra = make(map[string][]AckRumor)
	var mutex5 = sync.Mutex{}
	var racks = RumorAcks{
		racks:	ra,
		m:	mutex5,
	}
	
	var srt = make(map[string]bool)
	elementMap := make(map[string]bool)
	for i := 0; i < len(peers); i++ {
		elementMap[peers[i]] = true
	}
	var mutex4 = sync.Mutex{}
	var set_of_p = Set_of_peers{
		set: elementMap,
		m: mutex4,
	}

	var pendings []*PendingSearch
	var mutex2 = sync.Mutex{}
	ps := PendingSearches{
		ps: pendings,
		m: mutex2,
	}

	var mutex3 = sync.Mutex{}
	var pending_file_tab = make(map[string]*FileForSearch)
	var pending_files = PendingFiles{
		pf:	pending_file_tab,
		m:	mutex3,
	}
	
	var mutex6 = sync.RWMutex{}
	var s []PeerStatus
	var a []PeerMessage
	var rumor_state = Rumor_state{
		vector_clock:	s,
		archives:	a,
		m:	mutex6,
	}
	fmt.Println(elementMap)
	dsdv_state := make(map[string]string)
	var mutex7 = sync.Mutex{}
	var dsdv = DSDV{
		state:	dsdv_state,
		m:	mutex7,
	}

	var pa = make(map[string][]PrivateMessage)
	var mutex8 = sync.Mutex{}
	var archive_private = Private_state{
		archives:	pa,
		m:	mutex8,
	}


	var longest []*BlockWithLink
	var all_c = make(map[string][]*BlockWithLink)
	var all_b = make(map[string]*BlockWithLink)
	var filem = make(map[string][]byte)
	var already_seen = make(map[string]bool)
	var ptx []TxPublish
	var mutex9 = sync.RWMutex{}
	var blockchain = Blockchain{
		Longest: longest,
		All_c: all_c,
		All_b: all_b,
		already_seen: already_seen,
		FileMapping: filem,
		Pending: ptx,
		m: mutex9,
	}
	
	
	
	

	
	var lonelys	 = make(map[string]*Block)
	var mutex12 = sync.RWMutex{}
	var lonely_blocks = LonelyBlocks{
		lonelys: lonelys	,
		m: mutex12,
	}

	return &Gossiper{
		udp_address:        udpAddr,
		conn:               udpConn,
		Name:               name,
		set_of_peers:       set_of_p,
		rumor_state:	    rumor_state,
		archives_private:   archive_private,
		dsdv:               dsdv,
		simple:             simple,
		clientConn:         udpConnClient,
		clientAddr:         udpAddrClient,
		rtimer:             timer,
		file_pending:       pending_files,
		rumor_acks:         racks,
		search_req_timeout: srt,
		search_matches:     search_matches,
		pending_search:     ps,
		blockchain:			blockchain,
		lonely_blocks:		lonely_blocks,
	}
}

/**
This function return the UDPAddr of a peer chose randomly
*/
func (g *Gossiper) chooseRandomPeer() *net.UDPAddr {

	r := rand.Intn(len(g.set_of_peers.set))
	i := 0
	var addr *net.UDPAddr
	for k := range g.set_of_peers.set {
		if i == r {
			addr = ParseStrIP(k)
		}
		i++
	}

	return addr
}

/**
Convert a String in a *net.UDPAddr
*/
func ParseStrIP(str string) *net.UDPAddr {
	dst, err := net.ResolveUDPAddr("udp4", str)
	if err != nil {
		fmt.Println("error resolving the IP address")
		log.Fatal(err)
	}
	return dst
}

/**
Convert a *net.UDPAddr in a String
*/
func ParseIPStr(sender *net.UDPAddr) string {
	return sender.IP.String() + ":" + strconv.Itoa(sender.Port)
}

/**
Print rumor message that will be transmit to peers
*/
func printRumorMessageSnd(receiver *net.UDPAddr) {
	fmt.Println("MONGERING with " + ParseIPStr(receiver))
}

/**
Print StatusPacket received by me
*/
func printStatusMessageRcv(pkt *StatusPacket, sender *net.UDPAddr) {
	fmt.Print("STATUS from " + ParseIPStr(sender))
	for i := 0; i < len(pkt.Want); i++ {
		converted := fmt.Sprint(pkt.Want[i].NextID)
		fmt.Print(" peer " + pkt.Want[i].Identifier + " nextID " + converted)
	}
	fmt.Println("")
}

/**
Print that we flipped the coin and we will send message to a randomly choosed peer
*/
func flippedCoin(sender *net.UDPAddr) {
	fmt.Println("FLIPPED COIN sending rumor to " + ParseIPStr(sender))
}

/**
Print our neighbouring peers
*/
func (g *Gossiper) listAllKnownPeers() {
	fmt.Print("PEERS ")
	//mutex.Lock()
	for k := range g.set_of_peers.set {
		fmt.Print(k + ",")
	}
	fmt.Println("")
	//mutex.Unlock()
	return
}

/**
Print the rumor message that we received
*/
func printRumorMessageRcv(pkt *RumorMessage, sender *net.UDPAddr) {
	str := fmt.Sprint(pkt.ID)
	fmt.Println("RUMOR origin " + pkt.Origin + " from " + ParseIPStr(sender) + " ID " +
		str + " contents " + pkt.Text)
}

func printPrivateMessageRcv(pkt *PrivateMessage) {
	str := fmt.Sprint(pkt.HopLimit)
	fmt.Println("PRIVATE origin " + pkt.Origin + " hop-limit " + str + " contents " + pkt.Text)
}

func (g *Gossiper) printDSDV() {
	fmt.Println("DSDV :")

	for k, v := range g.dsdv.state {
		fmt.Println(k + " : " + v)
	}

}

func downloadPrint(filename string, chunk_nb int, origin string) {
	if chunk_nb >= 0 {
		str := fmt.Sprint(chunk_nb +1)
		fmt.Println("DOWNLOADING " + filename + " chunk " + str + " from " + origin)
	} else if chunk_nb == -1 {
		fmt.Println("RECONSTRUCTED file " + filename)
	} else if chunk_nb == -2 {
		fmt.Println("DOWNLOADING metafile of " + filename + " from " + origin)
	}
}

func searchPrint(filename string, origin string, metafile []byte, chunks []uint64) {
	fmt.Println("FOUND match " + filename + " at " + origin + " metafile=" + hex.EncodeToString(metafile) + " chunks=" + arrayToString(chunks, ","))

}

func printChain(bl *BlockWithLink){
	fmt.Print("CHAIN ")
	node := bl
	for node != nil{
		bl_hash := node.bl.Hash()
		fmt.Print(hex.EncodeToString(bl_hash[:]) +":"+hex.EncodeToString(node.bl.PrevHash[:])+":")
		first := true
		for _, tx := range node.bl.Transactions{
			if first {
				fmt.Print(tx.File.Name)
				first = false
			} else {
				fmt.Print("," + tx.File.Name)
			}	
		}
		fmt.Println("")
		node = node.prev
	}
}

func printFoundBlock(hash string){
	fmt.Println("FOUND-BLOCK " + hash)
}

func arrayToString(a []uint64, delim string) string {
	return strings.Trim(strings.Replace(fmt.Sprint(a), " ", delim, -1), "[]")
	//return strings.Trim(strings.Join(strings.Split(fmt.Sprint(a), " "), delim), "[]")
	//return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(a)), delim), "[]")
}

func printForkShorter(a [32]byte){
	fmt.Println("FORK-SHORTER "+hex.EncodeToString(a[:]))
}

func printForkLonger(a int){
	str := fmt.Sprint(a)
	fmt.Println("FORK-LONGER rewind "+str+" blocks")
}
