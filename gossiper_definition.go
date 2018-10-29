package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
)

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor  *RumorMessage
	Status *StatusPacket
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

	Name         string
	set_of_peers map[string]bool
	vector_clock []PeerStatus
	archives     []PeerMessage
	DSDV         map[string]string
	simple       bool
	clientConn   *net.UDPConn
	clientAddr   *net.UDPAddr
	rtimer       int
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

type UDPPacket struct {
	bytes   []byte
	err     error
	sender  *net.UDPAddr
	nb_byte int
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
	mutex.Lock()
	for k := range g.set_of_peers {
		fmt.Print(k + ",")
	}
	fmt.Println("")
	mutex.Unlock()
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

func (g *Gossiper) printDSDV() {
	fmt.Println("DSDV :")
	mutex.Lock()
	for k, v := range g.DSDV {
		fmt.Println(k + " : " + v)
	}
	mutex.Unlock()
}
