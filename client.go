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

	//log to file instead of console
	f := setLogClient()
	defer f.Close()

	server := auction.NewAuctionClient(connection) //creates a new client

	defer connection.Close()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		println("Enter 'bid' to make a new bid, or 'result' to retrieve the highest bid/outcome.")

		for {
			scanner.Scan()
			text := scanner.Text()

			if text == "bid" {
				println("Enter how much you would like to bid:")
				scanner.Scan()
				text := scanner.Text()
				bidAmount, err := strconv.Atoi(text)
				if err != nil {
					log.Fatal("We needed a int")
				}

				bid := &auction.SetBid{
					Amount:          int32(bidAmount),
					HighestBidderId: port,
				}

				ack := RecBid(bid, connection, server, port)

				log.Printf("Client %s: Bid response: ", port, ack.GetAcknowledgement())
				println("Bid response: ", ack.GetAcknowledgement())
			} else if text == "result" {

				getResult := &auction.GetResult{}

				result := RecResult(getResult, connection, server, port)
				// fmt.Println("result inside client: ", result)
				// if result == nil {
				// 	continue
				// }

				outcomeString := strconv.FormatInt(int64(result.Outcome), 10)
				log.Printf("Client %s: "+result.Message+". The result of the auction is: "+outcomeString, port)
				println(result.Message + ". The result of the auction is " + outcomeString + " and this bid was made by client" + result.HighestBidderId)
			} else {
				println("Sorry didn't catch that, try again ")
			}
		}
	}()

	for {

	}

}

func RecBid(setBid *auction.SetBid, connection *grpc.ClientConn, server auction.AuctionClient, port string) *auction.AckBid {
	ack, err := server.Bid(context.Background(), setBid)

	if err != nil {
		log.Printf("Client %s: Bid failed: ", port, err)
		log.Printf("Client %s: FEServer has died", port)
		connection, server = Redial(port)
		ack = RecBid(setBid, connection, server, port)

	}
	return ack

}

func RecResult(getResult *auction.GetResult, connection *grpc.ClientConn, server auction.AuctionClient, port string) *auction.ReturnResult {
	result, err := server.Result(context.Background(), getResult)
	if err != nil {
		log.Printf("Client %s: Bid failed: ", port, err)
		log.Printf("Client %s: FEServer has died", port)
		connection, server = Redial(port)
		result = RecResult(getResult, connection, server, port)
	}
	return result
}

func Redial(port string) (*grpc.ClientConn, auction.AuctionClient) {
	log.Printf("Client: FEServer on port %s is not listening anymore. It has died", port)
	if port == ":4001" {
		port = ":4002"
	} else {
		port = ":4001"
	}
	log.Printf("Client: Redialing to new port: " + port)
	connection, err := grpc.Dial(port, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Unable to connect: %v", err)
	}

	server = auction.NewAuctionClient(connection) //creates a new client
	log.Printf("Client: Client has connected to new FEServer on port %s", port)
	return connection, server
}

// sets the logger to use a log.txt file instead of the console
func setLogClient() *os.File {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}
