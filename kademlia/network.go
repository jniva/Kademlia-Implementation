package kademlia

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
	proto "github.com/golang/protobuf/proto"
	"bufio"
	"io"
	"os"
)

// Network the kademlia network
type Network struct {
	//routingTable *RoutingTable
	pingTable    *PingTable
	Kad     *Kademlia
	Contact *Contact
}
//Holds a UDP connection
type response struct {
	servr *net.UDPConn
	resp  *net.UDPAddr
}

//Holds packet information as JSON
type data struct {
	Rpc      string    `json:"rpc,omitempty"`
	Id       string    `json:"id,omitempty"`
	Ip       string    `json:"ip,omitempty"`
	Port	 string		`json:"port,omitempty"`
	Contacts []Contact `json:"data,omitempty"`
}

// NewNetwork create new network with bootstrap contact
func NewNetwork(port string, bootstrapAddress string, bootstrapPort string) *Network {
	network := &Network{}
	network.pingTable = NewPingTable()

	var nodeID = NewRandomKademliaID()
	fmt.Println("Random Node ID Generated: " + nodeID.String())
	contact := NewContact(nodeID, network.getLocalIP(), port)
	fmt.Println("Contact Generated: " + contact.ID.String() + " " + contact.Address+":"+contact.Port)

	//network.Kad.routingTable = NewRoutingTable(contact)
	network.Contact = &contact
	network.Kad = InitKad(contact)
	fmt.Println(network)

	bootstrapContact := NewContact(NewRandomKademliaID(), bootstrapAddress, bootstrapPort)
	fmt.Println("Bootsratp Contact: " + contact.ID.String() + " " + contact.Address)
	fmt.Println(bootstrapAddress)
	fmt.Println(bootstrapContact)
	fmt.Println(bootstrapPort)
	go SendPingMessage(&bootstrapContact, &contact)
	//go network.sendPingMessage(&bootstrapContact)
	return network
}

// get local IP from network interfaces
// This won't give external IP
func (network *Network) getLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return ""
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String()
		}
	}
	return ""
}

// Me get self contact from routing table
func (network *Network) Me() *Contact {
	//return &(network.routingTable.me)
	return  &(network.Kad.Rtable.me)
}

func (network *Network) createPacket() *Packet {
	packet := &Packet{}
	packet.Origin = &Node{}
	packet.Origin.KademliaId = network.Me().ID.String()
	packet.Origin.Address = network.Me().Address
	packet.Origin.Port = network.Me().Port
	return packet
}

//Listen to incoming requests
func (network *Network) Listen(port string) {
	//parsedPort, err := strconv.Atoi(port)
	//if err != nil {
	//	log.Fatal(err)
	//}

	fmt.Println("Starting to Listent on port: " + port)
	listenAdrs, err := net.ResolveUDPAddr("udp", network.getLocalIP()+":"+port)
	connection, err := net.ListenUDP("udp", listenAdrs)
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()
	msg := data{}
	for {
		msgbuf := make([]byte, 65536)
		n, resp, err := connection.ReadFromUDP(msgbuf)
		ErrorHandler(err)
		//size, address, err := connection.ReadFrom(data)
		//addr := strings.Split(address.String(), ":")
		json.Unmarshal(msgbuf[:n], &msg)
		fmt.Println(msg)
		Response := &response{
			servr: connection,
			resp:resp,
		}
		fmt.Println("resp: ", resp.IP)
		fmt.Println("Got Message:", msg)
		//go network.handleReceive(data, size, addr[0], addr[1], err)
		go network.rpcHandle(msg, *Response)
	}
}


func (network *Network) sendDataToAddress(address string, port string, data []byte) {
	fmt.Println("SendDatatoAddress: ", address,":",port)
	udpAddr, err := net.ResolveUDPAddr("udp", address+":"+port)
	if err != nil {
		fmt.Println(err)
	}
	connection, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer connection.Close()

	fmt.Println("sending packet data to " + address + ":" + port)

	_, err = connection.Write(data)
	if err != nil {
		fmt.Println("Error sending data to " + address + ":" + port)
		fmt.Println(err)
	}
	fmt.Println("Wrote a packet of data")
}

func (network *Network) processPacket(packet *Packet) {
	if packet.Origin != nil {
		network.handleOriginMessage(packet.Origin) // Do synchronously to prevent race conditions
	} else {
		fmt.Println("Received blank ORIGIN")
	}

	if packet.Ping != nil {
		go network.handlePingMessage(packet.Ping)
	}

	if packet.Pong != nil {
		go network.handlePongMessage(packet.Pong)
	}
}

func (network *Network) handleOriginMessage(origin *Node) {
	fmt.Println("Origin message from: " + origin.KademliaId + "(" + origin.Address + ":" + origin.Port + ")")
	if !network.Me().ID.Equals(NewKademliaID(origin.KademliaId)) {
		fmt.Println("Added contact")
		//network.routingTable.AddContact(NewContact(NewKademliaID(origin.KademliaId), origin.Address, origin.Port))
		network.Kad.Rtable.AddContact(NewContact(NewKademliaID(origin.KademliaId), origin.Address, origin.Port))

	} else {
		fmt.Println("Received origin message from self. Won't add.")
	}
	//result := network.routingTable.FindClosestContacts(NewKademliaID(origin.KademliaId), 20)
	result := network.Kad.Rtable.FindClosestContacts(NewKademliaID(origin.KademliaId), 20)

	for i := range result {
		fmt.Println("Has close contact " + result[i].ID.String() + " at " + result[i].Address)
	}
}

// PING RPC

// sendPingMessage send ping to the contact
func (network *Network) sendPingMessage(contact *Contact) {
	fmt.Println("sendPingMessage Contact:", contact.Address)
	network.sendPingMessageWithReplacement(contact, nil, nil, nil)
}

// sendPingMessageWithReplacement send ping to the contact,
// if contact responds then onResponse is called,
// otherwise onTimeout is called to replace the contact
func (network *Network) sendPingMessageWithReplacement(contact *Contact, replacement *Contact, onTimeout func(*KademliaID, *Contact, *Network), onResponse func(*Contact)) {
	fmt.Println("sendPingMessageWithReplacement Contact:", contact.Address)
	requestID := NewRandomKademliaID().String()
	fmt.Println("sendPingMessageWithReplacement RequestID:", requestID)
	network.pingTable.Push(requestID, contact.ID, onTimeout, onResponse)
	go network.handlePingTimeout(requestID, contact, replacement)
	network.sendPingPacket(requestID, contact)
}

func (network *Network) createPingPacket(requestID string) *Ping {
	ping := &Ping{}
	ping.RequestId = requestID
	ping.KademliaId = network.Me().ID.String()
	ping.Address = network.Me().Address
	ping.Port = network.Me().Port
	return ping
}

func (network *Network) sendPingPacket(requestID string, contact *Contact) {
	packet := network.createPacket()
	packet.Ping = network.createPingPacket(requestID)

	marshaledPacket, err := proto.Marshal(packet)
	if err != nil {
		fmt.Println("Error marshalling ping packet")
	}

	network.sendDataToAddress(contact.Address, contact.Port, marshaledPacket)
}

func (network *Network) handlePingTimeout(requestID string, old *Contact, replacement *Contact) {
	time.Sleep(time.Duration(2) * time.Second)
	pingRequest := network.pingTable.Pop(requestID)

	if pingRequest != nil {
		fmt.Println("Ping Request: " + requestID + " timed out.")
		if replacement == nil {
			fmt.Println("No replacement was found for ping request.")
		} else {
			fmt.Println("Replacement for ping request: " + replacement.ID.String())
			if pingRequest.onTimeout != nil {
				go pingRequest.onTimeout(old.ID, replacement, network)
			}
		}
	} else {
		fmt.Println("Ping Request: " + requestID + " received response in time.")
	}
}

func (network *Network) handlePingMessage(pingMessage *Ping) {
	fmt.Println("Ping message from: " + pingMessage.KademliaId)
	contact := NewContact(NewKademliaID(pingMessage.KademliaId), pingMessage.Address, pingMessage.Port)

	if !contact.ID.Equals(network.Me().ID) {
		//network.routingTable.AddContact(contact)
		network.Kad.Rtable.AddContact(contact)

		fmt.Println("Added " + pingMessage.KademliaId + " @ " + pingMessage.Address + " in bucket")
	} else {
		fmt.Println("Recieved self PING")
	}

	network.sendPongMessage(network.createPongMessage(pingMessage), pingMessage.Address, pingMessage.Port)
}

func (network *Network) createPongMessage(pingMessage *Ping) *Pong {
	pong := &Pong{}
	pong.RequestId = pingMessage.RequestId
	pong.KademliaId = network.Me().ID.String()
	pong.Address = network.Me().Address
	pong.Port = network.Me().Port
	return pong
}

func (network *Network) sendPongMessage(pongMessage *Pong, address string, port string) {
	packet := network.createPacket()
	packet.Pong = pongMessage
	out, err := proto.Marshal(packet)
	if err != nil {
		fmt.Println("Error marshaling Pong packet")
	} else {
		fmt.Println("Sending PONG for request :" + pongMessage.RequestId + "to " + address + ":" + port)
		network.sendDataToAddress(address, port, out)
	}

}

func (network *Network) handlePongMessage(pongMessage *Pong) {
	pingRequest := network.pingTable.Pop(pongMessage.RequestId)
	var contact Contact
	if pingRequest == nil {
		fmt.Println("Received pong with request id " + pongMessage.RequestId + " but not found in the ping table")
		contact = NewContact(NewKademliaID(pongMessage.KademliaId), pongMessage.Address, pongMessage.Port)
		if !network.Me().ID.Equals(contact.ID) {
			//network.routingTable.AddContact(contact)
			network.Kad.Rtable.AddContact(contact)

		}
	} else {
		contact = NewContact(pingRequest.kademliaID, pongMessage.Address, pongMessage.Port)
		if pingRequest.onResponse != nil {
			go pingRequest.onResponse(&contact)
		} else {
			if !network.Me().ID.Equals(contact.ID) {
				//network.routingTable.AddContact(contact)
				network.Kad.Rtable.AddContact(contact)

			}
		}
	}

	fmt.Println("PONG message from: " + pingRequest.kademliaID.String() + " for request ID: " + pongMessage.RequestId)
}

///// Command line interface functions:

func (network *Network) CliHelper(input io.Reader) {
	for {
		network.Cli(input)
	}
}
func (network *Network) Cli(input io.Reader) {

	cli := bufio.NewScanner(input)
	fmt.Printf("Command: \n")
	cli.Scan()
	text := cli.Text()
	fmt.Println(text)

	switch {
	case strings.Contains(text, "PUT "):
		storeData := []byte(text[4:])
		fmt.Println("Storing data on other nodes", storeData)
		network.IterativeStore(storeData)

	case strings.Contains(text, "GET "):
		hashData := text[4:]
		fmt.Println("Fetching data...")
		if len(hashData) == 40 {
			network.IterativeFindData(hashData)
			fmt.Println("The length of hash is correct.")
		} else {
			fmt.Println("The length of hash is wrong.")
		}

	case text == "EXIT":
		fmt.Println("Node is shutting down in 3 seconds...")
		time.Sleep(3 * time.Second)
		os.Exit(0)

	//Must ping an address
	case strings.Contains(text, "PING "):
		fmt.Println("Ping message is called")
		node := NewContact(NewRandomKademliaID(), text[5:], "8000")
		fmt.Println("Bootsratp Contact: " + node.ID.String() + " " + node.Address)
		//go network.sendPingMessage(&node)
		//node := NewContact(nil, text[5:])
		//msg, err := SendPingMessage(&node, network.Contact)
		//if err == nil {
		//	network.rpcHandle(msg, response{})
		//}

	//Must store a 3 characters to a given IP
	case strings.Contains(text, "STORE "):
		fmt.Println("Store is called...")
		storeData := []byte(text[6:9])
		fmt.Println(storeData)
		fmt.Println(text[10:])
		node := NewContact(nil, text[10:],"8000")
		SendStoreMessage(&node, storeData)

	case text == "CONTACTS":
		fmt.Println(network.Kad.Rtable.FindClosestContacts(NewKademliaID("0000000000000000000000000000000000000000"),160))
		for _, i := range network.Kad.Rtable.FindClosestContacts(NewKademliaID("0000000000000000000000000000000000000000"), 160) {
			fmt.Println("IN FOR LOOP OF CONTACTS",i.Address)
		}

	default:
		fmt.Println("default case")
		fmt.Sprintln("CLI not recognized")
	}
}

////CLI functions ended.

// error handling

func ErrorHandler(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

//creates the content of any message
func createMsg(rpc string, contact *Contact, c []Contact) *data {
	fmt.Println(contact.Address)
	fmt.Println(contact.Port)

	msg := &data{
		Rpc:      rpc,
		Id:       contact.ID.String(),
		Ip:       contact.Address,
		Port: 	  contact.Port,
		Contacts: c,
	}
	return msg
}

//Helper function to call LookupContact()
func (network *Network) IterativeFindNode(target *Contact) []Contact {
	result := make(chan []Contact)
	go network.Kad.LookupContact(network.Contact, result, *target)
	done := <-result
	fmt.Printf("\nIterativeFindNode done, found %d contacts \n", len(done))
	return done
}

//Helper function to call Store()
func (network *Network) IterativeStore(data []byte) {
	fmt.Println("Storing data on other nodes inside itereativeStore")
	network.Kad.Store(data, network.Contact)
}

//Helper function to call LookupData()
func (network *Network) IterativeFindData(hash string) {
	result := network.Kad.LookupData(network.Contact, NewContact(NewKademliaID(hash), "", "8000"), hash)
	if result[:2] == "OK" {
		fmt.Println("Value found: " + result[4:])
	} else {
		fmt.Println("Value not found. \nK closest contacts: " + result)
	}

}

//refreshes all buckets further away than closest neighbor
func (network *Network) updateBuckets() {
	//Loop over any populated bucket
	bucketPopulated := false
	//String reprensatation of bits in kadid
	sArr := make([]string, IDLength)
	//Loop over each byte
	for i := range [IDLength]int{} {
		//byte to bits
		bits := strconv.FormatInt(int64(network.Contact.ID[i]), 2)
		for j := len(bits); j < 8; j++ {
			bits = "0" + bits
		}
		sArr[i] = bits
	}
	//Start at LSByte
	for i := IDLength - 1; i >= 0; i-- {
		//Flip each bit at current byte
		for j := 7; j >= 0; j-- {
			fliped := sArr
			if string(fliped[i][j]) == "1" {
				fliped[i] = fliped[i][:j] + string("0") + fliped[i][j+1:]
			} else {
				fliped[i] = fliped[i][:j] + string("1") + fliped[i][j+1:]
			}

			toBytearr := BitsToKademliaID(fliped)
			//Update bucket at newID
			newID := NewKademliaID(hex.EncodeToString(toBytearr[:IDLength]))
			c := NewContact(newID, "","8000")
			if bucketPopulated == true || network.Kad.Rtable.buckets[network.Kad.Rtable.GetBucketIndex(newID)].Len() != 0 {
				arr := network.IterativeFindNode(&c)
				for _, c := range arr {
					network.Kad.Rtable.AddContact(c)
				}
				bucketPopulated = true
			}
		}
	}
}


//Calls functions by RPC value
func (network *Network) rpcHandle(msg data, r response) {
	switch {
	case strings.ToLower(msg.Rpc) == "ping":
		fmt.Println("ping received: ", msg)
		network.HandlePingMsg(msg, r)
	case strings.ToLower(msg.Rpc) == "pong":
		fmt.Println("pong received: ", msg)
		network.HandlePongMsg(msg)
	case strings.ToLower(msg.Rpc) == "find_node":
		fmt.Println("inside swite of rpchandle: fine_node rpc ")
		network.HandleFindNodeMsg(msg, r)
	case strings.ToLower(msg.Rpc) == "store":
		network.HandleStoreMsg(msg.Id, r)
	case strings.ToLower(msg.Rpc) == "find_value":
		network.HandleFindDataMsg(msg.Id, r)
	default:
		fmt.Println("Unknown RPC: " + msg.Rpc)
	}
}

//func (network *Network) handleReceive(data []byte, size int, addr string, port string, err error) {
//	if err != nil {
//		fmt.Println(err)
//	}
//
//	fmt.Println("Received packet from " + addr + ":" + port)
//
//	packet := &Packet{}
//	err = proto.Unmarshal(data[:size], packet)
//	if err != nil {
//		fmt.Println("Received an error from the ping command")
//		fmt.Println(err)
//	}
//
//	network.processPacket(packet)
//}


func (network *Network) HandleFindNodeMsg(msg data, r response) {
	Response := &data{
		Contacts: network.Kad.Rtable.FindClosestContacts(NewKademliaID(msg.Id), 20),
	}
	m, err := json.Marshal(Response)
	_, err = r.servr.WriteToUDP(m, r.resp)
	ErrorHandler(err)
	fmt.Println("SENT: my contacts")

}

func (network *Network) HandleFindDataMsg(msg string, r response) {
	if value, ok := network.Kad.hashmap[msg]; !ok {
		closeContactsArr := network.Kad.Rtable.FindClosestContacts(NewKademliaID(msg), 20)
		_, err := r.servr.WriteToUDP(ContactToByte(closeContactsArr), r.resp)
		ErrorHandler(err)
	} else {
		reply := []byte("OK: " + string(value))
		_, err := r.servr.WriteToUDP(reply, r.resp)
		ErrorHandler(err)
	}
}

func (network *Network) HandleStoreMsg(msg string, resp response) {
	hashedData := HashData([]byte(msg))
	fmt.Println(hashedData)

	if _, ok := network.Kad.hashmap[hashedData]; !ok {
		network.Kad.hashmap[hashedData] = []byte(msg)
		reply := []byte("File succesfully stored " + hashedData)
		_, err := resp.servr.WriteToUDP(reply, resp.resp)
		ErrorHandler(err)
	} else {
		reply := []byte(msg + " " + ("File already stored"))
		_, err := resp.servr.WriteToUDP(reply, resp.resp)
		ErrorHandler(err)

	}
}

//handles incoming ping msgs
func (network *Network) HandlePingMsg(msg data, r response) {
	contact := NewContact(NewKademliaID(msg.Id), msg.Ip,"8000")
	network.Kad.Rtable.AddContact(contact)
	SendPongMessage(r, network.Contact)
}

func (network *Network) HandlePongMsg(msg data) {
	fmt.Println("handle pong message", msg)
	fmt.Println(msg.Ip)
	contact := NewContact(NewKademliaID(msg.Id), msg.Ip,msg.Port)
	network.Kad.Rtable.AddContact(contact)
	fmt.Println("Added ponger: " + contact.Address)
}
