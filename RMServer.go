package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	auction "github.com/suvihanninen/DistributedAuctionSystem/grpc"
	"google.golang.org/grpc"
)

type RMServer struct {
	auction.UnimplementedAuctionServer
	id         int32
	peers      map[int32]auction.AuctionClient
	isPrimary  bool
	ctx        context.Context
	highestBid int32
	time       time.Time
	primary    auction.AuctionClient
}

func main() {
	portInput, _ := strconv.ParseInt(os.Args[1], 10, 32) //Takes arguments 0, 1 and 2, see comment X
	primary, _ := strconv.ParseBool(os.Args[2])
	ownPort := int32(portInput) + 5001

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rmServer := &RMServer{
		id:         ownPort,
		peers:      make(map[int32]auction.AuctionClient),
		ctx:        ctx,
		isPrimary:  primary,
		highestBid: 0,
		time:       time.Now().Local().Add(time.Second * time.Duration(120)),
		primary:    nil,
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
		if port == 5001 {
			rmServer.primary = c
		}
	}

	// If server is primary, dial to all other replica managers
	if rmServer.isPrimary {

	} else {
		go func() {
			for {
				time.Sleep(2 * time.Second)
				heartbeatMsg := &auction.Request{Message: "alive?"}
				response, err := rmServer.primary.GetHeartBeat(rmServer.ctx, heartbeatMsg)
				if err != nil {
					log.Printf("Something went wrong while sending heartbeat to:")
					log.Printf("Error:", err)
					delete(rmServer.peers, 5001)
					rmServer.ElectLeader()
				}
				log.Printf("We got a heart beat from %s", response)
				fmt.Println("IsPrimary:", rmServer.isPrimary)
				if rmServer.isPrimary {
					break
				}
			}
		}()
		for {

		}
		// implement dialing here if we have to implement heartbeats
	}
	for {

	}

}

func (RM *RMServer) ElectLeader() {
	fmt.Printf("INSIDE ELECT LEADER")
	var min int32
	min = RM.id
	for id := range RM.peers {
		fmt.Println("id: ", id)
		fmt.Println("min: ", min)
		if min > id {
			min = id
			fmt.Printf("Id was smaller than min")
		}
	}
	fmt.Printf("Smallest port %v", min)
	if RM.id == min {
		RM.isPrimary = true
	} else {
		RM.primary = RM.peers[min]
	}
}

func (RM *RMServer) GetHeartBeat(ctx context.Context, Heartbeat *auction.Request) (*auction.BeatAck, error) {
	return &auction.BeatAck{Port: fmt.Sprint(RM.id)}, nil
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
	if time.Now().After(rm.time) {
		return "Time is out", nil
	}
	if amount > rm.highestBid {
		rm.highestBid = amount
		updatedBid := &auction.SetBid{Amount: int32(amount)}

		//Broadcasting updated bid to all replica managers
		for id, server := range rm.peers {
			//Reconsider how to handle a potentially crashed replica manager
			ack, err := server.Bid(rm.ctx, updatedBid)
			if err != nil {
				log.Printf("Something went wrong when updating bid to %v", id)
				delete(rm.peers, id)
				numOfFailures++
			}

			log.Printf("Bid was updated to replica manager with ID: %s", ack)
		}
	} else {
		return "Failure", nil
	}

	if numOfFailures > 1 {
		return "Exception", nil
	}

	return "Success", nil
}

func (RM *RMServer) Result(ctx context.Context, GetResult *auction.GetResult) (*auction.ReturnResult, error) {
	message := ""
	if time.Now().After(RM.time) {
		message = "Time is out"
	} else {
		message = "The auction is ongoing"
	}
	println("Outcome inside Result: ", RM.highestBid)
	return &auction.ReturnResult{Outcome: RM.highestBid, Message: message}, nil
}
