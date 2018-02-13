/*Package craas implements a card reader TCP service.*/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var (
	// flagtesting drops us into testing mode, enabling a REPL to
	// input card reads
	flagtest = flag.Bool("testing", false, "input 'card reads' into a repl; useful for testing")
	// flagserial reads from the serial port
	flagserial = flag.String("serial", "", "serial port")
	// flagport publish to the host and port
	flagport = flag.String("port", ":8080", "network host and port")
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
type publisher struct {
	sync.Mutex                // lock em down
	rdr         io.Reader     // source of card events
	subscribers []chan []byte // things that are interested in the card events
}

func (pub *publisher) handle(conn io.WriteCloser) {
	c := make(chan []byte)
	defer conn.Close()

	// Add the new client channel into
	// the cardAuth struct
	func() {
		pub.Lock()
		defer pub.Unlock()
		pub.subscribers = append(pub.subscribers, c)
	}()

	for {
		// Listen for messages on the channel
		msg, closed := <-c
		if closed {
			return
		}

		buf := bytes.NewBuffer(msg)
		io.Copy(conn, buf)
	}

}

// publish moves buffers from the reader into the
// subscriber's channels
func (pub *publisher) publish() {
	bs := [256]byte{}
	for {
		n, err := pub.rdr.Read(bs[:])

		if err != nil {
			log.Println("Error on reader:", err)
			log.Println("The service is shutting down")

			func() {
				pub.Lock()
				defer pub.Unlock()
				for _, sub := range pub.subscribers {
					close(sub)
				}
			}()

			os.Exit(1)
		}

		func() {
			pub.Lock()
			defer pub.Unlock()

			removals := make([]int, 0)

			for idx, sub := range pub.subscribers {
				select {
				case sub <- bs[:n]:
					continue
				case <-time.After(200 * time.Millisecond):
					log.Println("subscriber channel failed to accept message")
					log.Println("dropping client...")
					removals = append(removals, idx)
				}
			}

			for removal := range removals {
				sub := pub.subscribers[removal]
				close(sub)
				pub.subscribers = append(pub.subscribers[:removal], pub.subscribers[removal+1:]...)
			}

		}()
	}
}

func main() {
	if !validateFlags() {
		os.Exit(1)
	}

	lis, err := net.Listen("tcp", *flagport)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("TCP service started at", *flagport)

	var rdr io.Reader
	if *flagtest {
		rdr = newRepl(">>")
	}

	pub := &publisher{rdr: rdr, subscribers: []chan []byte{}}
	go pub.publish()

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Fatalln("error accepting connection:", err)
			go pub.handle(conn)
		}
	}
}
