# Kademlia-Implementation
The objective of this is to produce a working Distributed Data Store (DDS). In contrast to a traditional database, a DDS stores its data objects on many different computers rather than only one. These computers together make up a single storage network, which internally keeps track of what objects are stored and which nodes keep copies of them. The system will be able to store and locate data objects by their hashes in such a storage network, which is formed by many running instances of your system. 

Simple network with nodes that can PING eachother
To build and run:
```
/bin/bash ./build.sh
```
It will spin the number of nodes (kademlia_nodes) mentioned in the file `build.sh`


```
docker-compose up --scale kademlia_nodes=3
```
In another terminal,
```
docker container ls
```
it will show you the list of containers running.

you can attach to specific container by
```
docker attach container_name
```
You will attach to the CLI running in the container. This will wait for inputs to send `PING`, `GET`, `STORE`, or `EXIT` commands

Examples:

```
PING IP_of_Node
```
```
PUT data
```
```
GET data
```
```
STORE data IP-Of_Node
```

## References:

- Udemy Courses for docker and goLang.
- GoLand docs
- http://blog.notdot.net/2009/11/Implementing-a-DHT-in-Go-part-1
- https://pub.tik.ee.ethz.ch/students/2006-So/SA-2006-19.pdf
- A book about kademlia development.
    https://github.com/SyncfusionSuccinctlyE-Books/The-Kademlia-Protocol-Succinctly
- For understanding Kademlia with the help of visualization:
    https://www.cs.helsinki.fi/u/sklvarjo/kademlia.html
    https://kelseyc18.github.io/kademlia_vis/basics/1/

###Thanks