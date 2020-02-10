docker rm $(docker ps -a -q) --force
docker image prune --force
docker build . -t kademlia
docker-compose up --scale kademlia_nodes=3