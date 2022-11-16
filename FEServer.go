package main

import (
	"context"
	"log"
	"net"

	"os"

	auction "github.com/suvihanninen/DistributedAuctionSystem/grpc"
	"google.golang.org/grpc"
)

type FEServer struct {
	auction.UnimplementedAuctionServer        // You need this line if you have a server
	port                               string // Not required but useful if your server needs to know what port it's listening to

}

/*
Start listening to client
Make connection with primary server (Rn we assume that there will be one and it will be waiting for us to connect)
Listen request from the client
Send response to the client
Send reuest to Primary RMServer
Listen for respnse
*/
func main() {
	port := os.Args[1] //give it a port and input the same port to the client
	address := ":" + port
	list, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("Server on port %s: Failed to listen on port %s: %v", port, address, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}
	grpcServer := grpc.NewServer()

	server := &FEServer{

		port: os.Args[1],
	}

	auction.RegisterAuctionServer(grpcServer, server) //Registers the server to the gRPC server.

	log.Printf("Server on port %s: Listening at %v\n", port, list.Addr())

	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

func (FE *FEServer) Bid(ctx context.Context, SetBid *auction.SetBid) (*auction.AckBid, error) {

	return &auction.AckBid{Acknowledgement: "This is a acknowledgement"}, nil
}

func (FE *FEServer) Result(ctx context.Context, GetResult *auction.GetResult) (*auction.ReturnResult, error) {
	return &auction.ReturnResult{Outcome: 10}, nil
}
