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

type cardAuth struct {
	sync.Mutex
	rdr         io.Reader
	subscribers []chan []byte
}

func (ca *cardAuth) GetCardEvents(_ *pb.CardRequest, stream pb.CardReader_GetCardEventsServer) error {
	c := make(chan []byte)

	func() {
		ca.Lock()
		defer ca.Unlock()
		ca.subscribers = append(ca.subscribers, c)
	}()

	for msg := range c {
		resp := &pb.CardResponse{
			Card: &pb.Card{
				Id: string(msg),
			},
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	return nil
}

func publish(ca *cardAuth) {
	bs := make([]byte, 255)
	for {
		n, err := ca.rdr.Read(bs)

		if err != nil {
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
	publish(ca)

	pb.RegisterCardReaderServer(grpcServer, ca)
	grpcServer.Serve(lis)
}
