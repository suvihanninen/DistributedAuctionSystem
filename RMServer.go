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
	id              int32
	peers           map[int32]auction.AuctionClient
	isPrimary       bool
	ctx             context.Context
	highestBid      int32
	time            time.Time
	primary         auction.AuctionClient
	highestBidderId string
}

func main() {
	portInput, _ := strconv.ParseInt(os.Args[1], 10, 32) //Takes arguments 0, 1 and 2, see comment X
	primary, _ := strconv.ParseBool(os.Args[2])
	ownPort := int32(portInput) + 5001

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rmServer := &RMServer{
		id:              ownPort,
		peers:           make(map[int32]auction.AuctionClient),
		ctx:             ctx,
		isPrimary:       primary,
		highestBid:      0,
		time:            time.Now().Local().Add(time.Second * time.Duration(100)),
		primary:         nil,
		highestBidderId: "",
	}

	//log to file instead of console
	f := setLogRMServer()
	defer f.Close()

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
		log.Printf("RMServer %v: Trying to dial: %v\n", rmServer.id, port)
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
				_, err := rmServer.primary.GetHeartBeat(rmServer.ctx, heartbeatMsg)
				if err != nil {
					log.Printf("RMServer %v: Something went wrong while sending heartbeat", rmServer.id)
					log.Printf("RMServer %v: Error:", rmServer.id, err)
					log.Printf("RMServer %v: Exception, We did not get heartbeat back from Primary Replica with port %v. It has died, ", rmServer.id)
					delete(rmServer.peers, 5001)
					rmServer.ElectLeader()
				}
				//comment out if you wanna see the logging of heartbeat
				//log.Printf("We got a heart beat from %s", response)

				if rmServer.isPrimary {
					break
				}
			}
		}()
		for {

		}
	}
	for {

	}

}

func (RM *RMServer) ElectLeader() {
	log.Printf("RMServer %v: Leader election started with Bully Algorithm", RM.id)
	var min int32
	min = RM.id
	for id := range RM.peers {
		if min > id {
			min = id
		}
	}

	if RM.id == min {
		RM.isPrimary = true
	} else {
		RM.primary = RM.peers[min]
	}
	log.Printf("RMServer %v: New Primary Replica has port %v ", RM.id, min)
}

func (RM *RMServer) GetHeartBeat(ctx context.Context, Heartbeat *auction.Request) (*auction.BeatAck, error) {
	return &auction.BeatAck{Port: fmt.Sprint(RM.id)}, nil
}

func (RM *RMServer) Bid(ctx context.Context, SetBid *auction.SetBid) (*auction.AckBid, error) {
	outcome, err := RM.updateBidToRm(SetBid.GetAmount(), SetBid.GetHighestBidderId())
	if err != nil {
		log.Fatalf("Updating bid to Replica Managers failed inside RMServer: %s", err)
	}

	return &auction.AckBid{Acknowledgement: outcome}, nil
}

func (rm *RMServer) updateBidToRm(amount int32, bidderId string) (string, error) {

	if time.Now().After(rm.time) {
		return "Failure: Time is out", nil
	}
	if amount > rm.highestBid {

		println("Inside if updating amount")
		println("Highertbid", rm.highestBid)
		println("Amount", amount)
		rm.highestBidderId = bidderId
		println("HighestBidder", bidderId)
		rm.highestBid = amount
		updatedBid := &auction.SetBid{Amount: int32(amount), HighestBidderId: bidderId}

		//Broadcasting updated bid to all replica managers
		for id, server := range rm.peers {
			//Reconsider how to handle a potentially crashed replica manager
			ack, err := server.UpdateBid(rm.ctx, updatedBid)
			if err != nil {
				log.Printf("RMServer %v: Something went wrong when updating bid to %v", rm.id, id)
				log.Printf("RMServer %v: Exception, Replica Manager on port %v died", rm.id, id)
				delete(rm.peers, id)
			}

			log.Printf("RMServer %v: Bid updated on replica manager on port %s with ack: ", rm.id, id, ack)
		}
	} else {
		println("Failure: Highest bid", rm.highestBid)
		println("Failure: Amount", amount)
		return "Failure: Bid wasn't enough big", nil
	}

	return "Success: Highest bid updated", nil
}

func (RM *RMServer) UpdateBid(ctx context.Context, SetBid *auction.SetBid) (*auction.AckBid, error) {
	RM.highestBid = SetBid.Amount
	RM.highestBidderId = SetBid.HighestBidderId
	println("UpdateBid: Highest bid", RM.highestBid)
	log.Printf("RMServer %v: Replica Manager updated. Value: ", RM.id, RM.highestBid)
	outcome := "updated"
	return &auction.AckBid{Acknowledgement: outcome}, nil
}

func (RM *RMServer) Result(ctx context.Context, GetResult *auction.GetResult) (*auction.ReturnResult, error) {
	message := ""
	if time.Now().After(RM.time) {
		message = "Time is out"
	} else {
		message = "The auction is ongoing"
	}
	log.Printf("RMServer %v: Outcome inside Result: ", RM.id, RM.highestBid)
	return &auction.ReturnResult{Outcome: RM.highestBid, Message: message, HighestBidderId: RM.highestBidderId}, nil
}

// sets the logger to use a log.txt file instead of the console
func setLogRMServer() *os.File {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}
