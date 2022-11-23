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

type RMServer struct {
	auction.UnimplementedAuctionServer
	id           int32
	peers        map[int32]auction.AuctionClient
	isPrimary    bool
	ctx          context.Context
	highestBid int32
}

func main() {
	portInput, _ := strconv.ParseInt(os.Args[1], 10, 32) //Takes arguments 0, 1 and 2, see comment X
	primary, _ := strconv.ParseBool(os.Args[2])
	ownPort := int32(portInput) + 5001

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rmServer := &RMServer{
		id:           ownPort,
		peers:        make(map[int32]auction.AuctionClient),
		ctx:          ctx,
		isPrimary: 	primary,
		highestBid: 0,
	}

	//Primary needs to listen so that replica managers can ask if it's alive
	//Replica managers need to listen for incoming data to be replicated
	list, err := net.Listen("tcp", fmt.Sprintf(":%v", ownPort))
	if err != nil {
		log.Fatalf("Failed to listen on port: %v", err)
	}
	grpcServer := grpc.NewServer()
	auction.RegisterAuctionServer(grpcServer, rmServer)
	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to server %v", err)
		}
	}()

	// If server is primary, dial to all other replica managers
	if (rmServer.isPrimary) {
		for i := 0; i < 3; i++ {
			port := int32(5001) + int32(i)
	
			if port == ownPort {
				continue
			}
	
			var conn *grpc.ClientConn
			log.Printf("Trying to dial: %v\n", port)
			conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock()) //This is going to wait until it receives the connection
			if err != nil {
				log.Fatalf("Could not connect: %s", err)
			}
			defer conn.Close()
			c := auction.NewAuctionClient(conn)
			rmServer.peers[port] = c
		}
	} else {
		for {}
		// implement dialing here if we have to implement heartbeats
	}
	for {}
	// if rmServer.isPrimary {

	// 	scanner := bufio.NewScanner(os.Stdin)
	// 	for {
	// 		log.Println("Enter a bid: ")
	// 		scanner.Scan()
	// 		bidText := scanner.Text()
	// 		bidNumber, _ := strconv.ParseInt(bidText, 10, 32)
	// 		bid := &auction.SetBid{
	// 			Amount: int32(bidNumber),
	// 		}
	// 		outcome, err := rmServer.Bid(rmServer.ctx, bid)
	// 		if err != nil {
	// 			log.Fatalf("bid failed inside main: %s", err)
	// 		}
	// 		log.Println("Outcome inside main: ", outcome)
	// 	}
	// }
	

}



func (RM *RMServer) Bid(ctx context.Context, SetBid *auction.SetBid) (*auction.AckBid, error) {
	outcome, err := RM.updateBidToRm(SetBid.GetAmount())
	if err != nil {
		log.Fatalf("updateBidToRm failed inside RMServer: %s", err)
	}
	println("Outcome inside bid: ", outcome)
	return &auction.AckBid{Acknowledgement: outcome}, nil
}

func (rm *RMServer) updateBidToRm(amount int32) (string, error) {
	numOfFailures := 0
	if amount > rm.highestBid {
		rm.highestBid = amount
		updatedBid := &auction.SetBid{Amount: int32(amount)}
		
		//Broadcasting updated bid to all replica managers
		for id, rmServer := range rm.peers {
			//Reconsider how to handle a potentially crashed replica manager
			ack, err := rmServer.Bid(rm.ctx, updatedBid)
			if err != nil {
				log.Printf("Something went wrong when updating bid to %v", id)
				delete(rm.peers, id)
				numOfFailures++
			}
			
			log.Printf("Bid was updated to replica manager with ID: %s", ack)
		} 		
	} else {
		return "failure", nil
	}

	if numOfFailures >1 {
		return "exception", nil
	}

	return "success", nil
}

func (RM *RMServer) Result(ctx context.Context, GetResult *auction.GetResult) (*auction.ReturnResult, error) {
	return &auction.ReturnResult{Outcome: 10}, nil
}