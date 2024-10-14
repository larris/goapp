package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sync"
)

func main() {
	numConn := flag.Int("n", 1, "Number of parallel connections")
	flag.Parse()

	var wg sync.WaitGroup

	// Handle interrupts for graceful shutdown
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Channel to signal all goroutines to stop
	done := make(chan struct{})

	for i := 0; i < *numConn; i++ {
		wg.Add(1)
		go func(connId int) {
			defer wg.Done()
			u := url.URL{
				Scheme: "ws",
				Host:   "localhost:8080",
				Path:   "/goapp/ws",
			}
			c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

			if err != nil {
				log.Fatalf("[Conn #%d] Failed to connect: %v", connId, err)
				return
			}
			defer c.Close()

			// Start reading messages
			for {
				select {
				case <-done:
					//  receive a signal to stop,
					log.Printf("[Conn #%d] Closing connection.", connId)
					return
				default:
					_, message, err := c.ReadMessage()
					if err != nil {
						log.Printf("[Conn #%d] Read error: %v", connId, err)
						return
					}
					log.Printf(string(message))
				}
			}

		}(i + 1) // Pass the connection ID starting from 1
	}

	// Wait for all connections to finish or interrupt signal
	<-interrupt
	log.Println("Received interrupt signal, closing connections...")
	close(done)
	wg.Wait()
	log.Println("All connections closed.")
}
