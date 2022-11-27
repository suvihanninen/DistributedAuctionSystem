package main

import (
	"context"
	"fmt"
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
	primaryServer                      auction.AuctionClient
	ctx                                context.Context
}

/*
Start listening to client
Make connection with primary server (Rn we assume that there will be one and it will be waiting for us to connect)
Listen request from the client
Send response to the client
Send reuest to Primary RMServer
Listen for respnse
*/
var serverToDial int

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
		port:          os.Args[1],
		primaryServer: nil,
		ctx:           ctx,
	}

	auction.RegisterAuctionServer(grpcServer, server) //Registers the server to the gRPC server.

	log.Printf("Server on port %s: Listening at %v\n", port, list.Addr())
	go func() {
		log.Printf("We are trying to listen calls from client: " + port)

		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}

		log.Printf("We have started to listen calls from client: " + port)
	}()

	serverToDial = 5001
	conn := server.DialToPR(serverToDial)

	defer conn.Close()

	for {
	}

}

func (FE *FEServer) DialToPR(serverToDial int) *grpc.ClientConn {
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
	outcome, err := FE.primaryServer.Bid(ctx, SetBid)
	if err != nil {
		log.Println("bid failed inside FEServer: %s", err)
		//if we get an error we need to Dial to another port
		FE.Redial("bid", SetBid.GetAmount())
	}
	log.Println("Outcome inside FEserver: ", outcome)
	return outcome, nil
}

func (FE *FEServer) Result(ctx context.Context, GetResult *auction.GetResult) (*auction.ReturnResult, error) {
	log.Println("Inside FEServer Result getting outcome")
	log.Println("FE primary server: ")
	outcome, err := FE.primaryServer.Result(ctx, GetResult)
	if err != nil {
		log.Printf("result fetching failed inside FEServer: %s", err)
		//if we get an error we need to Dial to another port
		FE.Redial("result", 0)
		return nil, nil
	}

	fmt.Println("outcome1: ", outcome)

	return outcome, nil
}

func (FE *FEServer) Redial(functionType string, bid int32) string {
	log.Println("inside redial for function: ", functionType)
	// portNumber, err := strconv.ParseInt(FE.port, 10, 32)

	// if err != nil {
	// 	log.Printf("Error while parsing string to int: ", err)
	// }
	portNumber := int64(serverToDial) + int64(1)

	fmt.Printf("New PR port: ", portNumber)
	FE.DialToPR(int(portNumber))
	//redial
	if functionType == "result" {
		getResult := &auction.GetResult{}
		outcome, _ := FE.Result(FE.ctx, getResult)
		return outcome.Message
	} else { //is bid
		setBid := &auction.SetBid{
			Amount: bid,
		}
		outcome, _ := FE.Bid(FE.ctx, setBid)
		return outcome.Acknowledgement
	}
}
