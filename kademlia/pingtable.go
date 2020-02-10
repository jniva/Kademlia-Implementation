package kademlia

// Ping calls and their timeouts implementation

import (
	"sync"
)

// PingRequest stores a requestID and Kademlia ID of the item in question,
// bucket ID can be derived from it.
type PingRequest struct {
	requestID  string
	kademliaID *KademliaID
	onTimeout  func(*KademliaID, *Contact, *Network)
	onResponse func(*Contact)
}

// PingTable a PING should make an entry in this table
// and THEN send PING over network with TIMEOUT
// If response is received then delete the entry from the table
// and update the bucket
type PingTable struct {
	pingRequests []PingRequest
	lock         *sync.Mutex
}

// NewPingTable create new Ping table for the network
func NewPingTable() *PingTable {
	table := &PingTable{}
	table.pingRequests = make([]PingRequest, 2*bucketSize)
	table.lock = &sync.Mutex{}
	return table
}

// Push a PING with timeout and response handlers
func (table *PingTable) Push(requestID string, kademliaID *KademliaID, onTimeout func(*KademliaID, *Contact, *Network), onResponse func(*Contact)) {
	table.lock.Lock()
	defer table.lock.Unlock()
	table.pingRequests = append(table.pingRequests, PingRequest{requestID, kademliaID, onTimeout, onResponse})
}

// Pop remove a Ping from the Table and returns it or nil if not found
func (table *PingTable) Pop(requestID string) *PingRequest {
	table.lock.Lock()
	defer table.lock.Unlock()
	for i := 0; i < len(table.pingRequests); i = i + 1 {
		if table.pingRequests[i].requestID == "" {
			continue
		}
		if table.pingRequests[i].requestID == requestID {
			item := table.pingRequests[i]

			table.pingRequests[i] = table.pingRequests[len(table.pingRequests)-1]          // Copy last element to index i
			table.pingRequests[len(table.pingRequests)-1] = PingRequest{"", nil, nil, nil} // Erase last element (write zero value)
			table.pingRequests = table.pingRequests[:len(table.pingRequests)-1]            // Truncate slice

			return &item
		}
	}
	return nil
}
