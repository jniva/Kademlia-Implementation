package kademlia

import (
	"encoding/hex"
	"fmt"
	//"strconv"

	"github.com/google/uuid"
)

// the static number of bytes in a KademliaID
const IDLength = 20

// type definition of a KademliaID
type KademliaID [IDLength]byte

// NewKademliaID returns a new instance of a KademliaID based on the string input
func NewKademliaID(data string) *KademliaID {
	decoded, _ := hex.DecodeString(data)

	newKademliaID := KademliaID{}
	for i := 0; i < IDLength; i++ {
		newKademliaID[i] = decoded[i]
	}

	return &newKademliaID
}

// NewRandomKademliaID returns a new instance of a random KademliaID using UUID
func NewRandomKademliaID() *KademliaID {
	newKademliaID := KademliaID{}
	copy(newKademliaID[:], uuid.New().String())
	fmt.Println("Random Node ID Generated inside NewRandomKademliaID: ",newKademliaID )
	return &newKademliaID
}

// Less returns true if kademliaID < otherKademliaID (bitwise)
func (kademliaID KademliaID) Less(otherKademliaID *KademliaID) bool {
	for i := 0; i < IDLength; i++ {
		if kademliaID[i] != otherKademliaID[i] {
			return kademliaID[i] < otherKademliaID[i]
		}
	}
	return false
}

// Equals returns true if kademliaID == otherKademliaID (bitwise)
func (kademliaID KademliaID) Equals(otherKademliaID *KademliaID) bool {
	for i := 0; i < IDLength; i++ {
		if kademliaID[i] != otherKademliaID[i] {
			return false
		}
	}
	return true
}

// CalcDistance returns a new instance of a KademliaID that is built
// through a bitwise XOR operation betweeen kademliaID and target
func (kademliaID KademliaID) CalcDistance(target *KademliaID) *KademliaID {
	result := KademliaID{}
	for i := 0; i < IDLength; i++ {
		result[i] = kademliaID[i] ^ target[i]
	}
	return &result
}

// String returns a simple string representation of a KademliaID
func (kademliaID *KademliaID) String() string {
	return hex.EncodeToString(kademliaID[0:IDLength])
}
