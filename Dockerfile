FROM larjim/kademlialab:latest

RUN mkdir /home/go/src/app
COPY . /home/go/src/app
COPY kademlia /home/go/src/kademlia
WORKDIR /home/go/src/app
ENV GOPATH /home/go
ENV PATH="${GOPATH}/bin:${PATH}"
RUN /usr/local/go/bin/go get github.com/google/uuid
RUN protoc --go_out=../kademlia *.proto
RUN CGO_ENABLED=0 GOOS=linux GOARCH=386 /usr/local/go/bin/go build -o main .
ENTRYPOINT ["sh","./run.sh"]