package kademlia

import (
	"testing"
)

func TestPingTable(t *testing.T) {
	table := NewPingTable()
	rID := NewRandomKademliaID().String()
	kID := NewRandomKademliaID()
	notFound := NewRandomKademliaID().String()
	table.Push(rID, kID, nil, nil)
	elem := table.Pop(rID)
	if elem == nil {
		t.Error("Expected to find element added to table")
	}
	elem = table.Pop(notFound)
	if elem != nil {
		t.Error("Did not expect to find 'notFound' in table")
	}
}
