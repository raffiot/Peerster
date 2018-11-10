package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
)


type GossipPacket struct {
	Simple  *SimpleMessage
	Rumor   *RumorMessage
	Status  *StatusPacket
	Private *PrivateMessage
	DataRequest *DataRequest
    DataReply   *DataReply
}

type ClientPacket struct {
	Simple  *SimpleMessage
	Private *PrivateMessage
	File 	*FileMessage
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

	Name             string
	set_of_peers     map[string]bool
	vector_clock     []PeerStatus
	archives         []PeerMessage
	archives_private map[string][]PrivateMessage
	DSDV             map[string]string
	simple           bool
	clientConn       *net.UDPConn
	clientAddr       *net.UDPAddr
	rtimer           int
	file_pending	 map[string][]File
	rumor_acks		 map[string][]AckRumor
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
	Filename 	string
	Request		string
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

type File struct{
	Filename	string
	Filesize 	int
	Metafile	[]byte
	Metahash 	[]byte
}

type DataRequest struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
}

type DataReply struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
	Data []byte
}

type AckRumor struct {
	Identifier 	string
	NextID 		uint32
	ch			chan bool
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

	var s []PeerStatus
	var a []PeerMessage
	var pa = make(map[string][]PrivateMessage)
	var pending_file_tab = make(map[string][]File)
	var ra = make(map[string][]AckRumor)
	elementMap := make(map[string]bool)
	for i := 0; i < len(peers); i++ {
		elementMap[peers[i]] = true
	}
	fmt.Println(elementMap)
	dsdv := make(map[string]string)
	return &Gossiper{
		udp_address:      udpAddr,
		conn:             udpConn,
		Name:             name,
		set_of_peers:     elementMap,
		vector_clock:     s,
		archives:         a,
		archives_private: pa,
		DSDV:             dsdv,
		simple:           simple,
		clientConn:       udpConnClient,
		clientAddr:       udpAddrClient,
		rtimer:           timer,
		file_pending:	  pending_file_tab,
		rumor_acks:		  ra,
	}
}



/**
This function return the UDPAddr of a peer chose randomly
*/
func (g *Gossiper) chooseRandomPeer() *net.UDPAddr {

	r := rand.Intn(len(g.set_of_peers))
	i := 0
	var addr *net.UDPAddr
	for k := range g.set_of_peers {
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
	for k := range g.set_of_peers {
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

	for k, v := range g.DSDV {
		fmt.Println(k + " : " + v)
	}

}

func downloadPrint(filename string, chunk_nb int, origin string){
	if chunk_nb >= 0 {
		str := fmt.Sprint(chunk_nb)
		fmt.Println("DOWNLOADING "+filename+ " chunk "+str+ " from " + origin)
	} else if chunk_nb == -1 {
		fmt.Println("RECONSTRUCTED file "+filename)
	} else if chunk_nb == -2 {
		fmt.Println("DOWNLOADING metafile of "+ filename+" from "+origin)
	}
}



