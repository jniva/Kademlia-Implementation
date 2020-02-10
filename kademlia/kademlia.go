package kademlia

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	//"log"
	"strings"

	//"crypto/sha1"
	//"encoding/hex"
	"math/rand"
	"net"
	"time"

	//"net"
	//"strings"
)

type Kademlia struct {
	Rtable  *RoutingTable
	hashmap map[string][]byte
}
type Shortlist struct {
	ls []Contact
	v  map[string]bool
}

func InitKad(me Contact) *Kademlia {
	node := &Kademlia{
		Rtable:  NewRoutingTable(me),
		hashmap: make(map[string][]byte),
	}
	return node
}

const a = 3

func (kademlia *Kademlia) LookupContact(me *Contact, result chan []Contact, target Contact) {
	// TODO
	fmt.Println("inside lookupcontact")
	fmt.Println(me.String())
	fmt.Println(target.String())

	var closestNode Contact
	var x []Contact
	found := make(chan []Contact)
	doublet := make(map[string]bool)
	iRoutines := 0
	myClosest := kademlia.Rtable.FindClosestContacts(target.ID, a)
	closestNode = myClosest[0]
	sl := &Shortlist{
		ls: make([]Contact, 0),
		v:  make(map[string]bool),
	}

	for _, mine := range myClosest {
		sl.insert(false, mine)
		doublet[mine.ID.String()] = true
	}

	for iRoutines < a && iRoutines < len(sl.ls) {
		fmt.Println(sl.ls[iRoutines])
		go SendFindContactMessage(&sl.ls[iRoutines], found, sl, &target)
		x = <-found
		sl.v[sl.ls[iRoutines].ID.String()] = true
		iRoutines++
	}

	for iRoutines > 0 {
		recived := x
		for _, candidate := range recived {
			if !(candidate.Address == me.Address) && !(candidate.ID == nil) {
				if doublet[candidate.ID.String()] == false {
					doublet[candidate.ID.String()] = true
					candidate.CalcDistance(me.ID)
					sl.insert(false, candidate)
				}
			}
		}
		sl.ls = qsort(sl.ls, target)
		iRoutines--
		if closestNode.ID.String() != sl.ls[0].ID.String() {
			closestNode = sl.ls[0]
			for i := range sl.ls {
				if i >= a || i >= len(sl.ls) {
					break
				}
				if sl.v[sl.ls[i].ID.String()] == false {
					iRoutines++
					go SendFindContactMessage(&sl.ls[i], found, sl, &target)
					x = <-found
					sl.v[sl.ls[i].ID.String()] = true
				}

			}
		}
	}
	for i, c := range sl.ls {
		if i >= len(sl.ls) {
			break
		}
		if sl.v[sl.ls[i].ID.String()] == false {
			go SendFindContactMessage(&c, found, sl, &target)
			x = <-found
			sl.v[c.ID.String()] = true
		}
	}
	sl.ls = qsort(sl.ls, target)
	if len(sl.ls) > 20 {
		result <- sl.ls[:20]
	} else {
		result <- sl.ls
	}
}

func (kademlia *Kademlia) LookupData(me *Contact, target Contact, hash string) string {
	alpha := 3
	value := make(chan string)
	found := make(chan []Contact)
	var x []Contact
	var y string
	myClosest := kademlia.Rtable.FindClosestContacts(target.ID, alpha)
	var shortlist []Contact
	var noKeyShortlist []Contact
	doublet := make(map[string]bool)
	visited := make(map[string]bool)
	for _, mine := range myClosest {
		shortlist = append(shortlist, mine)
		doublet[mine.ID.String()] = true
	}

	runningRoutines := 0
	for runningRoutines < 3 && len(shortlist) > 1 {
		go SendFindDataMessage(hash, &shortlist[runningRoutines], found, value)
		x = <-found
		y = <-value
		if y != "" {
			if len(noKeyShortlist) > 0 {
				fmt.Println("Storing at closest contact")
				SendStoreMessage(&noKeyShortlist[0], []byte(y))
			}
			runningRoutines = 0
			return y
		}
		runningRoutines++
		for _, i := range x {
			if i.ID != nil {
				i.CalcDistance(target.ID)
				noKeyShortlist = append(noKeyShortlist, i)
			}
		}
		noKeyShortlist = qsort(noKeyShortlist, target)
	}

	if len(shortlist) == 1 {
		runningRoutines++
		go SendFindDataMessage(hash, &shortlist[0], found, value)
		x = <-found
		y = <-value
		if y != "" {
			if len(noKeyShortlist) > 0 {
				fmt.Println("Storing at closest contact")
				SendStoreMessage(&noKeyShortlist[0], []byte(y))
			}
			runningRoutines = 0
			return y
		}
		for _, i := range x {
			if i.ID != nil {
				i.CalcDistance(target.ID)
				noKeyShortlist = append(noKeyShortlist, i)
			}
		}
		noKeyShortlist = qsort(noKeyShortlist, target)

	}

	for runningRoutines > 0 && len(x) > 0 {
		recived := x
		for _, candidate := range recived {
			if !(candidate.Address == me.Address) && !(candidate.ID == nil) {
				if doublet[candidate.ID.String()] == false {
					doublet[candidate.ID.String()] = true
					candidate.CalcDistance(target.ID)
					shortlist = append(shortlist, candidate)

				}
			}
		}
		shortlist = qsort(shortlist, target)
		runningRoutines--
		for i := range shortlist {

			if visited[shortlist[i].ID.String()] == false {
				visited[shortlist[i].ID.String()] = true
				runningRoutines++
				go SendFindDataMessage(hash, &shortlist[i], found, value)
				x = <-found
				y = <-value
				if y != "" {
					if len(noKeyShortlist) > 0 {
						fmt.Println("Storing at closest contact")
						SendStoreMessage(&noKeyShortlist[0], []byte(y))
					}
					runningRoutines = 0
					return y
				}
				for _, i := range x {
					if i.ID != nil {
						i.CalcDistance(target.ID)
						noKeyShortlist = append(noKeyShortlist, i)
					}
				}
				noKeyShortlist = qsort(noKeyShortlist, target)
			}
		}
	}

	shortlist = qsort(shortlist, target)

	var shortlistString string

	if len(shortlist) > 20 {
		for _, i := range shortlist[:20] {
			shortlistString = shortlistString + i.String() + "\n"
		}
	} else {
		for _, i := range shortlist {
			shortlistString = shortlistString + i.String() + "\n"
		}
	}

	return shortlistString
}


func (kademlia *Kademlia) Store(data []byte, me *Contact) {
	ch := make(chan []Contact)
	fmt.Println("Storing data on other nodes inside kad.store")
	contact := NewContact(NewKademliaID(HashData(data)), me.Address, me.Port)
	go kademlia.LookupContact(me, ch, contact)
	done := <-ch
	for _, c := range done {
		SendStoreMessage(&c, data)
	}
}

// shortList interface
func (sl *Shortlist) insert(v bool, c Contact) []Contact {
	sl.ls = append(sl.ls, c)
	return sl.ls
}
func (sl *Shortlist) removeContact(c Contact) {
	for i, f := range sl.ls {
		if f.ID.String() == c.ID.String() {
			copy(sl.ls[i:], sl.ls[i+1:])
			sl.ls = sl.ls[:len(sl.ls)-1]
			return
		}

	}
	fmt.Println("contact not in list")
}

// sorting
func qsort(contact []Contact, target Contact) []Contact {
	if len(contact) < 2 {
		return contact
	}

	left, right := 0, len(contact)-1

	pivot := rand.Int() % len(contact)

	contact[pivot], contact[right] = contact[right], contact[pivot]

	for i := range contact {
		dist := contact[i].ID.CalcDistance(target.ID)
		distr := contact[right].ID.CalcDistance(target.ID)
		if dist.Less(distr) {
			contact[left], contact[i] = contact[i], contact[left]
			left++
		}
	}

	contact[left], contact[right] = contact[right], contact[left]

	qsort(contact[:left], target)
	qsort(contact[left+1:], target)

	return contact
}

// send find contact message for lookupcontact
func SendFindContactMessage(contact *Contact, found chan []Contact, sl *Shortlist, target *Contact) {
	fmt.Println("inside sendfindcontactmessage")
	fmt.Println(contact.String())
	fmt.Println(contact.Port)
	fmt.Println(contact.Address)

	RemoteAddress, err := net.ResolveUDPAddr("udp", contact.Address+":"+contact.Port)
	fmt.Println("remote address: ", RemoteAddress)
	connection, err := net.DialUDP("udp", nil, RemoteAddress)
	//connection.SetDeadline(time.Now().Add(50 * time.Millisecond))
	ErrorHandler(err)
	defer connection.Close()
	msg, err := json.Marshal(createMsg("find_node", target, nil))
	ErrorHandler(err)
	_, err = connection.Write(msg)
	fmt.Println("SENT: " + string(msg) + " to: " + contact.Address+":"+contact.Port)
	respmsg := make([]byte, 65536)
	data := data{}
	//connection.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
	var c []Contact
	for {
		n, err := connection.Read(respmsg)
		if err != nil {
			if e, ok := err.(net.Error); !ok && !e.Timeout() {
				ErrorHandler(e)
				c = make([]Contact, 0)
				break
			}
			fmt.Println("Node offline!")
			sl.removeContact(*contact)
			c = make([]Contact, 0)
			break
		}
		err = json.Unmarshal(respmsg[:n], &data)
		ErrorHandler(err)
		c = data.Contacts
		break
	}
	found <- c
}

//for lookupdata
func SendFindDataMessage(hash string, contact *Contact, found chan []Contact, value chan string) {
	RemoteAddress, err := net.ResolveUDPAddr("udp", contact.Address+":"+contact.Port)
	connection, err := net.DialUDP("udp", nil, RemoteAddress)
	ErrorHandler(err)
	defer connection.Close()
	d := &data{Id: hash, Rpc: "find_value"}
	msg, err := json.Marshal(d)
	_, err = connection.Write(msg)
	ErrorHandler(err)

	respmsg := make([]byte, 65536)
	n, err := connection.Read(respmsg)
	ErrorHandler(err)

	c := make([]Contact, 0)
	if string(respmsg[:2]) == "OK" {
		found <- c
		value <- string(respmsg[:n])
	} else {
		c = ByteToContact(respmsg[:n])
		found <- c
		value <- ""
	}
}

func SendStoreMessage(contact *Contact, b []byte) {
	RemoteAddress, err := net.ResolveUDPAddr("udp", contact.Address+":"+contact.Port)
	connection, err := net.DialUDP("udp", nil, RemoteAddress)
	ErrorHandler(err)
	defer connection.Close()
	Response := &data{Id: string(b), Rpc: "store"}
	m, err := json.Marshal(Response)
	_, err = connection.Write(m)
	ErrorHandler(err)
	respmsg := make([]byte, 65536)
	n, err := connection.Read(respmsg)
	ErrorHandler(err)
	fmt.Println(string(respmsg[:n]))
}

//
func SendPingMessage(contact *Contact, me *Contact) (data, error) {
	RemoteAddress, err := net.ResolveUDPAddr("udp", contact.Address+":"+contact.Port)
	//localAddress, err := net.ResolveUDPAddr("udp", me.Address+":"+me.Port)
	connection, err := net.DialUDP("udp", nil, RemoteAddress)
	ErrorHandler(err)
	defer connection.Close()
	fmt.Println(me.String())
	fmt.Println("object after creating Msg")
	fmt.Println(createMsg("ping", me, nil))
	msg, err := json.Marshal(createMsg("ping", me, nil))
	_, err = connection.Write(msg)
	ErrorHandler(err)
	fmt.Println("SENT: " + string(msg))
	data := data{}
	respmsg := make([]byte, 65536)
	connection.SetReadDeadline(time.Now().Add(1 * time.Second))
	for {
		n, err := connection.Read(respmsg)
		if err != nil {
			if e, ok := err.(net.Error); !ok && !e.Timeout() {
				ErrorHandler(e)
				break
			}
			fmt.Println("Node offline!")
			break
		}
		err = json.Unmarshal(respmsg[:n], &data)
		ErrorHandler(err)
		return data, nil

	}
	return data, err
}

func SendPongMessage(r response, me *Contact) {
	RemoteAddress, err := net.ResolveUDPAddr("udp", r.resp.IP.String()+":8000")
	connection, err := net.DialUDP("udp", nil, RemoteAddress)
	fmt.Println("resp inside sendpondmsg: ", r.resp.String())
	fmt.Println("resp inside sendpondmsg: ", r.resp.Port)
	fmt.Println(me.Address)
	fmt.Println(me.Port)
	msg, err := json.Marshal(createMsg("pong", me, nil))
	_, err = connection.Write(msg)
	ErrorHandler(err)
	defer connection.Close()
	fmt.Println("resp inside sendpondmsg: ", r.resp)
	//_, err = r.servr.WriteToUDP(msg, r.resp)
	//ErrorHandler(err)
	fmt.Println("SENT: " + string(msg))
	//fmt.Println("SENT: ", "pong ", me)
}

//for sendfinddatamessage
func ByteToContact(msg []byte) []Contact {
	s := string(msg)
	slice := strings.Split(s, "\n")
	arr := make([]Contact, 0)
	//var contact Contact
	for _, line := range slice {
		if len(line) != 0 {
			contact := NewContact(NewKademliaID(line[:40]), line[41:41+strings.Index(line[41:], "")],"8000")
			if len(line[41+strings.Index(line[41:], " "):]) > 2 {
				contact.distance = NewKademliaID(line[41+strings.Index(line[41:], " "):])
			}
			arr = append(arr, contact)
		}
	}
	return arr
}

//for store data
func HashData(data []byte) string {
	hashedData := sha1.Sum(data)
	hashedStringdata := hex.EncodeToString(hashedData[0:])
	return hashedStringdata
}

func ContactToByte(contactArr []Contact) []byte {
	closeCToByte := make([]byte, 0)
	closeContactsByte := make([]byte, 0)
	for i := 0; i < len(contactArr); i++ {
		closeCToByte = []byte(contactArr[i].ID.String() + " " + contactArr[i].Address + " " + contactArr[i].distance.String() + "\n")
		closeContactsByte = append(closeContactsByte, closeCToByte[:]...)
	}

	return closeContactsByte

}