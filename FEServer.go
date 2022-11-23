package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	auction "github.com/suvihanninen/DistributedAuctionSystem/grpc"
	"google.golang.org/grpc"
)

type FEServer struct {
	auction.UnimplementedAuctionServer        // You need this line if you have a server
	port                               string // Not required but useful if your server needs to know what port it's listening to
	primaryServer auction.AuctionClient
	ctx context.Context
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := &FEServer{
		port: os.Args[1],
		primaryServer: nil,
		ctx: ctx,
	}

	auction.RegisterAuctionServer(grpcServer, server) //Registers the server to the gRPC server.

	log.Printf("Server on port %s: Listening at %v\n", port, list.Addr())
	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()
	

	serverToDial := 5001
	conn := server.DialToPR(serverToDial)
	defer conn.Close()
	
	// scanner := bufio.NewScanner(os.Stdin)
	// for {
	// 	log.Println("Enter a bid: ")
	// 	scanner.Scan()
	// 	bidText := scanner.Text()
	// 	bidNumber, _ := strconv.ParseInt(bidText, 10, 32)
	// 	bid := &auction.SetBid{
	// 		Amount: int32(bidNumber),
	// 	}
	// 	outcome, err := server.primaryServer.Bid(server.ctx, bid)
	// 	if err != nil {
	// 		log.Fatalf("bid failed inside main: %s", err)
	// 	}
	// 	log.Println("Outcome inside main: ", outcome)
	// }
	for{}

}

func (FE *FEServer) DialToPR(serverToDial int) (*grpc.ClientConn) {
	//Dialing to the primary replica manager
	portToDial := ":" + strconv.Itoa(serverToDial)
	connection, err := grpc.Dial(portToDial, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect: %v", err)
	}
	log.Println("Connection established with PR.")
	primServer := auction.NewAuctionClient(connection)
	FE.primaryServer = primServer
	return connection
}

func (FE *FEServer) Bid(ctx context.Context, SetBid *auction.SetBid) (*auction.AckBid, error) {
	log.Println("bid inside FEserver: ", SetBid)
	outcome, err :=FE.primaryServer.Bid(ctx, SetBid)
	if err != nil {
		log.Fatalf("bid failed inside FEServer: %s", err)
	}
	log.Println("Outcome inside main: ", outcome)
	return outcome, nil
}

func (FE *FEServer) Result(ctx context.Context, GetResult *auction.GetResult) (*auction.ReturnResult, error) {
	return &auction.ReturnResult{Outcome: 10}, nil
}
