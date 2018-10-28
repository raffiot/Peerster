package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dedis/protobuf"
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
	simple       bool
	clientConn   *net.UDPConn
	clientAddr   *net.UDPAddr
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
func listAllKnownPeers(g *Gossiper) {
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

/**
Constructor of Gossiper
*/
func NewGossiper(address string, name string, peers []string, simple bool, client string) *Gossiper {
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

	elementMap := make(map[string]bool)
	for i := 0; i < len(peers); i++ {
		elementMap[peers[i]] = true
	}

	return &Gossiper{
		udp_address:  udpAddr,
		conn:         udpConn,
		Name:         name,
		set_of_peers: elementMap,
		vector_clock: s,
		archives:     a,
		simple:       simple,
		clientConn:   udpConnClient,
		clientAddr:   udpAddrClient,
	}
}

/**
Routine that handle the messages that we receive from the client
at <127.0.0.1:ClientPort>
We handle only the GossipPacket of type SimpleMessage
*/
func receiveMessageFromClient(g *Gossiper) {

	defer g.clientConn.Close()

	for {
		var pkt GossipPacket = GossipPacket{}
		b := make([]byte, UDP_PACKET_SIZE)
		nb_byte_written, _, err := g.clientConn.ReadFromUDP(b)
		protobuf.Decode(b, &pkt)
		if nb_byte_written > 0 && err == nil {
			if pkt.Simple != nil {
				handleSimplePacket(pkt.Simple, g)
			} else {
				fmt.Println("Error client send packet with type different than simple message.")
			}
		}
	}
}

/**
Function called when we receive a SimpleMessage from client
if we are in simple mode:
 	we send this SimpleMessage to all neighbors
if we aren't in simple mode:
	We take from the vector clock datastructure the ID of this new message
	We send this message as RumorMessage to the gossiper connexion
*/
func handleSimplePacket(pkt *SimpleMessage, g *Gossiper) {
	fmt.Println("CLIENT MESSAGE " + pkt.Contents)
	if g.simple {
		newPkt := GossipPacket{Simple: &SimpleMessage{
			OriginalName:  g.Name,
			RelayPeerAddr: ParseIPStr(g.udp_address),
			Contents:      pkt.Contents,
		}}

		pktByte, err := protobuf.Encode(&newPkt)
		if err != nil {
			fmt.Println("Encode of the packet failed")
			log.Fatal(err)
		}
		listAllKnownPeers(g)
		mutex.Lock()
		for k := range g.set_of_peers {
			dst, err := net.ResolveUDPAddr("udp4", k)
			if err != nil {
				fmt.Println("cannot resolve addr of ather gossiper")
				log.Fatal(err)
			}
			g.conn.WriteToUDP(pktByte, dst)
		}
		mutex.Unlock()
	} else {
		var my_ID uint32 = 1
		mutex.Lock()
		for i := 0; i < len(g.vector_clock); i++ {
			if g.vector_clock[i].Identifier == g.Name {
				my_ID = g.vector_clock[i].NextID
			}
		}
		mutex.Unlock()
		newPkt := GossipPacket{Rumor: &RumorMessage{
			Origin: g.Name,
			ID:     my_ID,
			Text:   pkt.Contents,
		}}
		pktByte, err := protobuf.Encode(&newPkt)
		if err != nil {
			fmt.Println("Encode of the packet failed")
			log.Fatal(err)
		}
		mutex.Lock()
		g.conn.WriteToUDP(pktByte, g.udp_address)
		mutex.Unlock()
	}
	return
}

/**
This function handle SimpleMessage from other gossipers
	if we don't know the sender we add him to our list of peers
	we send this message to our peers except the sender
*/
func handleSimplePacketG(pkt *SimpleMessage, sender *net.UDPAddr, g *Gossiper) {
	sender_formatted := ParseIPStr(sender)
	fmt.Println("SIMPLE MESSAGE origin " + pkt.OriginalName + " from " +
		sender_formatted + " contents " + pkt.Contents)

	newPkt := GossipPacket{Simple: &SimpleMessage{
		OriginalName:  pkt.OriginalName,
		RelayPeerAddr: ParseIPStr(g.udp_address),
		Contents:      pkt.Contents,
	}}

	//SEND PKT TO ALL
	packet_encoded, err := protobuf.Encode(&newPkt)
	if err != nil {
		fmt.Println("cannot encode message")
		log.Fatal(err)
	}
	listAllKnownPeers(g)
	mutex.Lock()
	for k := range g.set_of_peers {
		if k != sender_formatted {
			dst, err := net.ResolveUDPAddr("udp4", k)
			if err != nil {
				fmt.Println("cannot resolve addr of other gossiper")
				log.Fatal(err)
			}
			g.conn.WriteToUDP(packet_encoded, dst)
		}
	}
	mutex.Unlock()
}

/**
This function receive all packet on the gossiper interface.
it also implement the anti-entropy process
	if the sending peer is unknonw we add him to our list of peers
	if the packet is a SimpleMessage we call handleSimplePacketG
	if the packet is a StatusPacket we call StatusPacketRoutine
	if the packet is a RumorMessage we call rumorMessageRoutine
*/
func receiveMessageFromGossiper(g *Gossiper) {
	receivedInTime := false
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for _ = range ticker.C {
			if !receivedInTime {
				dst := chooseRandomPeer(g)
				sendMyStatus(dst, g, 0)
			}
		}
	}()

	defer g.conn.Close()
	for {
		var pkt GossipPacket = GossipPacket{}

		res := UDPPacket{}
		b := make([]byte, UDP_PACKET_SIZE)
		receivedInTime = false
		nb_byte_written, sender, err := g.conn.ReadFromUDP(b)
		receivedInTime = true
		res = UDPPacket{
			bytes:   b,
			err:     err,
			sender:  sender,
			nb_byte: nb_byte_written,
		}

		protobuf.Decode(res.bytes, &pkt)

		if res.nb_byte > 0 && res.err == nil {
			sender_formatted := ParseIPStr(res.sender)
			mutex.Lock()
			_, ok := g.set_of_peers[sender_formatted]
			if !ok && sender_formatted != ParseIPStr(g.udp_address) {
				g.set_of_peers[sender_formatted] = true
			}
			mutex.Unlock()
			if pkt.Simple != nil {
				handleSimplePacketG(pkt.Simple, res.sender, g)
			} else if pkt.Status != nil {
				StatusPacketRoutine(pkt.Status, res.sender, g)
			} else if pkt.Rumor != nil {
				if sender_formatted != ParseIPStr(g.udp_address) {
					printRumorMessageRcv(pkt.Rumor, res.sender)
					listAllKnownPeers(g)
				}
				rumorMessageRoutine(pkt.Rumor, res.sender, g)
			}
		}
	}
	return
}

/**
This function update both archives and vector clock with the arriving RumorMessage
it return two booleans
alreadyHave: true if we already have the pkt in our archives
notAdded: true if the packet has an id too high so we miss RumorMessage before that one
*/
func updateArchivesVC(pkt *RumorMessage, sender *net.UDPAddr, g *Gossiper) (bool, bool) {

	var containSender bool = false
	var alreadyHave bool = false
	var notAdded bool = false
	mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		var ps PeerStatus = g.vector_clock[i]
		if ps.Identifier == pkt.Origin {
			if ps.NextID == pkt.ID {
				g.archives[i].msgs[ps.NextID] = &RumorMessage{
					Origin: pkt.Origin,
					ID:     pkt.ID,
					Text:   pkt.Text,
				}
				g.vector_clock[i].NextID = ps.NextID + 1
			} else if ps.NextID > pkt.ID {
				alreadyHave = true
			} else {
				notAdded = true
			}
			containSender = true
		}
	}

	if !containSender {
		a := make(map[uint32]*RumorMessage)
		var nID uint32
		if pkt.ID == 1 {
			nID = 2
			a[uint32(1)] = &RumorMessage{
				Origin: pkt.Origin,
				ID:     uint32(1),
				Text:   pkt.Text,
			}
		} else {
			nID = 1
			notAdded = true
		}
		g.vector_clock = append(g.vector_clock, PeerStatus{
			Identifier: pkt.Origin,
			NextID:     nID,
		})

		g.archives = append(g.archives, PeerMessage{
			Identifier: pkt.Origin,
			msgs:       a,
		})
	}
	mutex.Unlock()
	return alreadyHave, notAdded
}

/**
This function is called to answer correctly to the received RumorMessage
we send our Status to the sender of the RumorMessage only if the sender is not ourselves
if the message interested us we monger it.
*/
func rumorMessageRoutine(pkt *RumorMessage, sender *net.UDPAddr, g *Gossiper) {

	alreadyHave, notAdded := updateArchivesVC(pkt, sender, g)
	if ParseIPStr(sender) != ParseIPStr(g.udp_address) {
		sendMyStatus(sender, g, 0)
	}

	if !alreadyHave && !notAdded {
		newDst := chooseRandomPeer(g)
		sendingRoutine(pkt, newDst, g)
	}

	return
}

/**
This function return the UDPAddr of a peer chose randomly
*/
func chooseRandomPeer(g *Gossiper) *net.UDPAddr {
	mutex.Lock()
	r := rand.Int() % len(g.set_of_peers)
	i := 0
	var addr *net.UDPAddr
	for k := range g.set_of_peers {
		if i == r {
			addr = ParseStrIP(k)
		}
		i++
	}
	mutex.Unlock()
	return addr
}

/**
This function send the rumor message to peer dst and wait for an ack from him
if we receive an ack we send the rumor to another random peer
*/
func sendingRoutine(pkt *RumorMessage, dst *net.UDPAddr, g *Gossiper) {

	pktNew := GossipPacket{Rumor: &RumorMessage{
		Origin: pkt.Origin,
		ID:     pkt.ID,
		Text:   pkt.Text,
	}}
	pktByte, err := protobuf.Encode(&pktNew)

	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}
	printRumorMessageSnd(dst)
	mutex.Lock()
	g.conn.WriteToUDP(pktByte, dst)
	mutex.Unlock()

	ch := make(chan *StatusPacket, 1)
	go receivePkt(ch, dst, g.conn)
	select {
	case res := <-ch:
		StatusPacketRoutine(res, dst, g)
	case <-time.After(1 * time.Second):
		rnd := rand.Int() % 2
		if rnd == 0 {
			newDst := chooseRandomPeer(g)
			flippedCoin(newDst)
			sendingRoutine(pkt, newDst, g)
		}
	}

	return
}

func receivePkt(out chan<- *StatusPacket, senderExpected *net.UDPAddr, conn *net.UDPConn) {
	pkt := GossipPacket{}
	b := make([]byte, UDP_PACKET_SIZE)
	nb_byte_written, sender, err := conn.ReadFromUDP(b)
	if nb_byte_written > 0 && err == nil {
		protobuf.Decode(b, &pkt)
		if pkt.Status != nil {
			if ParseIPStr(sender) == ParseIPStr(senderExpected) {
				out <- pkt.Status
			}
		}
	}
	return
}

/**
Function called when we receive a StatusPacket from another peer
numberToAsk represent the number of message that we don't already have
if numberToAsk > 0 then we send our status to the sender
if sender missed packet we send the Rumor message missed
*/
func StatusPacketRoutine(pkt *StatusPacket, sender *net.UDPAddr, g *Gossiper) {
	printStatusMessageRcv(pkt, sender)
	listAllKnownPeers(g)
	vco := pkt.Want
	var vcm []PeerStatus
	mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		p := PeerStatus{
			Identifier: g.vector_clock[i].Identifier,
			NextID:     g.vector_clock[i].NextID,
		}
		vcm = append(vcm, p)
	}
	mutex.Unlock()
	//vcm := g.vector_clock

	numberToAsk := 0

	for i := 0; i < len(vcm); i++ {
		var contains bool = false
		elemM := vcm[i]
		for j := 0; j < len(vco); j++ {
			elemO := vco[j]
			if elemM.Identifier == elemO.Identifier {
				contains = true
				if elemM.NextID > elemO.NextID {
					sendUpdate(elemO, sender, g)
				} else if elemM.NextID < elemO.NextID {
					numberToAsk += int(elemO.NextID) - int(elemM.NextID)
				}
			}
		}
		if !contains {
			p := PeerStatus{
				Identifier: elemM.Identifier,
				NextID:     1,
			}
			sendUpdate(p, sender, g)
		}
	}

	for i := 0; i < len(vco); i++ {
		elemO := vco[i]
		var contains bool = false
		for j := 0; j < len(vcm); j++ {
			elemM := vcm[j]
			if elemM.Identifier == elemO.Identifier {
				contains = true
			}
		}
		if !contains {
			numberToAsk += int(elemO.NextID) - 1
		}
	}
	if numberToAsk > 0 {
		sendMyStatus(sender, g, numberToAsk)
	} else {
		fmt.Println("IN SYNC WITH " + ParseIPStr(sender))
	}
	return
}

/**
Send our status and wait for numberToAsk messages to arrive so we are up to date
*/
func sendMyStatus(sender *net.UDPAddr, g *Gossiper, numberToAsk int) {
	var w []PeerStatus
	mutex.Lock()
	for i := 0; i < len(g.vector_clock); i++ {
		w = append(w, PeerStatus{
			Identifier: g.vector_clock[i].Identifier,
			NextID:     g.vector_clock[i].NextID,
		})
	}
	mutex.Unlock()
	newPkt := GossipPacket{Status: &StatusPacket{
		Want: w,
	}}

	pktByte, err := protobuf.Encode(&newPkt)
	if err != nil {
		fmt.Println("Encode of the packet failed")
		log.Fatal(err)
	}
	mutex.Lock()
	g.conn.WriteToUDP(pktByte, sender)
	mutex.Unlock()

	for i := 0; i < numberToAsk; i++ {
		pkt := GossipPacket{}
		b := make([]byte, UDP_PACKET_SIZE)
		nb_byte_written, sender, err := g.conn.ReadFromUDP(b)
		if nb_byte_written > 0 && err == nil {
			protobuf.Decode(b, &pkt)
			if pkt.Rumor != nil {
				updateArchivesVC(pkt.Rumor, sender, g)
			}
		}
	}
}

/**
We send the rumor messages from a specific origin the sender doesn't have yet
*/
func sendUpdate(pDefault PeerStatus, sender *net.UDPAddr, g *Gossiper) {

	mutex.Lock()
	my_message := g.archives
	for i := 0; i < len(my_message); i++ {
		if my_message[i].Identifier == pDefault.Identifier {
			var finished bool = false
			for j := pDefault.NextID; !finished; {
				rm, ok := my_message[i].msgs[j]
				if ok {

					newPkt := GossipPacket{Rumor: &RumorMessage{
						Origin: rm.Origin,
						ID:     rm.ID,
						Text:   rm.Text,
					}}

					pktByte, err := protobuf.Encode(&newPkt)
					if err != nil {
						fmt.Println("Encode of the packet failed")
						log.Fatal(err)
					}
					g.conn.WriteToUDP(pktByte, sender)
					j++
				} else {
					finished = true
				}
			}
		}
	}
	mutex.Unlock()
	return
}

var UDP_PACKET_SIZE = 2048
var me *Gossiper
var mutex sync.Mutex

func main() {
	client_ip := "127.0.0.1"
	uiport := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper (default \"127.0.0.1:5000\")")
	name := flag.String("name", "", "name of the gossiper")
	peers := flag.String("peers", "", "coma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	server := flag.Bool("server", false, " run the server")
	flag.Parse()

	me = NewGossiper(*gossipAddr, *name, strings.Split(*peers, ","), *simple, client_ip+":"+*uiport)

	go receiveMessageFromClient(me)
	if *server {
		go receiveMessageFromGossiper(me)
		http.HandleFunc("/message", MessageHandler)
		http.HandleFunc("/node", NodeHandler)
		http.HandleFunc("/id", IdHandler)

		if err := http.ListenAndServe("localhost:8080", nil); err != nil {
			panic(err)
		}
	} else {
		receiveMessageFromGossiper(me)
	}

}

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	if r.Method == "GET" {
		var values []RumorMessage
		w.WriteHeader(http.StatusOK)
		mutex.Lock()
		for i := 0; i < len(me.archives); i++ {
			finished := false
			j := 1
			for !finished {
				elem, ok := me.archives[i].msgs[uint32(j)]
				j++
				if ok {
					values = append(values, *elem)
				} else {
					finished = true
				}
			}
		}
		mutex.Unlock()
		jsonValue, _ := json.Marshal(values)
		w.Write(jsonValue)
	} else if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			panic(err)
		}

		pkt_to_enc := GossipPacket{Simple: &SimpleMessage{
			OriginalName:  "client",
			RelayPeerAddr: "",
			Contents:      r.Form.Get("value"),
		}}
		packetBytes, err := protobuf.Encode(&pkt_to_enc)
		me.clientConn.WriteToUDP(packetBytes, me.clientAddr)
	}

}

func NodeHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	mutex.Lock()
	values := me.set_of_peers
	mutex.Unlock()
	jsonValue, _ := json.Marshal(values)
	w.Write(jsonValue)
}

func IdHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.WriteHeader(http.StatusOK)
	values := me.Name
	jsonValue, _ := json.Marshal(values)
	w.Write(jsonValue)
}
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
