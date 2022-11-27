package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strconv"

	auction "github.com/suvihanninen/DistributedAuctionSystem/grpc"
	"google.golang.org/grpc"
)

var server auction.AuctionClient //the server
var ServerConn *grpc.ClientConn  //the server connection

func main() {
	/*
		we want to bid
		query the auction
		save the most resent result
		If client looses the connection to a FEServer we need to Dial to the next one
	*/
	port := ":" + os.Args[1] //we give a portnumber where it can dial to
	connection, err := grpc.Dial(port, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect: %v", err)
	}

	server := auction.NewAuctionClient(connection) //creates a new client

	ServerConn = connection
	defer ServerConn.Close()

	defer connection.Close()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		log.Println("Enter 'bid' to make a new bid, or 'result' to retrieve the highest bid/outcome.")

		for {
			scanner.Scan()
			text := scanner.Text()

			if text == "bid" {
				log.Println("Enter how much you would like to bid:")
				scanner.Scan()
				text := scanner.Text()
				bidAmount, err := strconv.Atoi(text)
				if err != nil {
					log.Fatal("We needed a int")
				}

				bid := &auction.SetBid{
					Amount: int32(bidAmount),
				}

				ack, err := server.Bid(context.Background(), bid)
				if err != nil {
					log.Printf("Bid failed:")
					log.Println(err)
					log.Println("We need to handle crash")
					Dial(port)
				}

				log.Println("Bid response: ", ack.Acknowledgement)

			} else if text == "result" {

				getResult := &auction.GetResult{}

				result, err := server.Result(context.Background(), getResult)
				if err != nil {
					log.Printf("Result failed:")
					log.Println(err)
					Redial(port)
				}
				outcomeString := strconv.FormatInt(int64(result.Outcome), 10)
				log.Println(result.Message + ". The result of the auction is: " + outcomeString)

			} else {
				log.Println("Sorry didn't catch that, try again ")
			}
		}
	}()

	for {

	}

}

func Redial(port string) *grpc.ClientConn {
	log.Printf("Port which is not listening anymore: " + port)
	if port == ":4001" {
		port = ":4002"
	} else {
		port = ":4001"
	}

	connection, err := grpc.Dial(port, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect: %v", err)
	}

	server = auction.NewAuctionClient(connection) //creates a new client

	ServerConn = connection
	log.Printf("Client has connected to port %s", port)
	return ServerConn
}

func HandleCrash() {

}
