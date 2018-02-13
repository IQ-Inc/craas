/*Package client demonstrates a golang card reader consumer. The client will print
the messages received from the connected card reader service. You may spawn multiple
clients, and they'll all receive event notifications.

Change the target URL and / or port, then build and run:

	go build
	./client
*/
package main

import (
	"context"
	"log"

	pb "github.com/IQ-Inc/craas/craasbuf"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial(":8080", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	cardClient := pb.NewCardReaderClient(conn)
	cardReq := &pb.CardRequest{}
	stream, err := cardClient.GetCardEvents(context.Background(), cardReq)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Listening for card events...")
	for {
		msg, err := stream.Recv()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Received", msg.Card.Id)
	}
}
