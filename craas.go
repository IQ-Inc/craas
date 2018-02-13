package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	pb "github.com/IQ-Inc/craas/craasbuf"
	"google.golang.org/grpc"
)

var (
	// flagtesting drops us into testing mode, enabling a REPL to
	// input card reads
	flagtest = flag.Bool("testing", false, "input 'card reads' into a repl; useful for testing")
	// flagserial reads from the serial port
	flagserial = flag.String("serial", "", "serial port")
)

// validateFlags parsers and validates command-line flags. Returns false if
// there was something silly, else true
func validateFlags() bool {
	flag.Parse()

	if *flagtest && *flagserial != "" {
		fmt.Fprintln(os.Stderr, "error: cannot specify a serial port with testing")
		return false
	} else if !*flagtest && *flagserial == "" {
		fmt.Fprintln(os.Stderr, "error: no serial port provided")
		return false
	}

	return true
}

// cardAuth hodls the reader of card events.
// It pushes reader messages to the subscribers
type cardAuth struct {
	sync.Mutex                // lock em down
	rdr         io.Reader     // source of card events
	subscribers []chan []byte // things that are interested in the card events
}

func (ca *cardAuth) GetCardEvents(_ *pb.CardRequest, stream pb.CardReader_GetCardEventsServer) error {
	c := make(chan []byte)
	idx := -1

	// Add the new client channel into
	// the cardAuth struct
	func() {
		ca.Lock()
		defer ca.Unlock()

		idx = len(ca.subscribers)
		ca.subscribers = append(ca.subscribers, c)
	}()

	// Listen for messages on the channel
	for msg := range c {
		resp := &pb.CardResponse{
			Card: &pb.Card{
				Id: string(msg),
			},
		}

		if err := stream.Send(resp); err != nil {
			// Remove the subscriber from the list of subscribers
			func() {
				ca.Lock()
				defer ca.Unlock()
				ca.subscribers = append(ca.subscribers[:idx], ca.subscribers[idx+1:]...)
			}()
			return err
		}
	}

	return nil
}

// publish moves buffers from the reader into the
// subscriber's channels
func publish(ca *cardAuth) {
	bs := [256]byte{}
	for {
		n, err := ca.rdr.Read(bs[:])

		if err != nil {
			log.Println("Error on reader:", err)
			func() {
				ca.Lock()
				defer ca.Unlock()
				for _, sub := range ca.subscribers {
					close(sub)
				}
			}()
			return
		}

		func() {
			ca.Lock()
			defer ca.Unlock()
			for _, sub := range ca.subscribers {
				sub <- bs[:n]
			}
		}()
	}
}

func main() {
	if !validateFlags() {
		os.Exit(1)
	}

	var rdr io.Reader
	if *flagtest {
		rdr = newRepl(">>")
	}

	lis, err := net.Listen("tcp", ":8888")
	if err != nil {
		log.Fatalln(err)
	}

	grpcServer := grpc.NewServer()
	ca := &cardAuth{rdr: rdr, subscribers: []chan []byte{}}
	go publish(ca)

	pb.RegisterCardReaderServer(grpcServer, ca)
	grpcServer.Serve(lis)
}
