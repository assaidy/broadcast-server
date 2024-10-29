package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/websocket"
)

type WSServer struct {
	clients map[*websocket.Conn]struct{}
}

func NewWSServer() *WSServer {
	return &WSServer{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

func (wss *WSServer) handleClient(c *websocket.Conn) {
	log.Printf("client connected: %s", c.RemoteAddr().String())
	wss.clients[c] = struct{}{}
	wss.startReading(c)
}

func (wss *WSServer) startReading(c *websocket.Conn) {
	buf := make([]byte, 1024*2)
	for {
		n, err := c.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("error reading message from client: %s", c.RemoteAddr().String())
			continue
		}
		msg := fmt.Sprintf("[%s]: %s", c.RemoteAddr().String(), buf[:n])
		wss.broadcast([]byte(msg))
	}
	log.Printf("client disconnected: %s", c.RemoteAddr().String())
	delete(wss.clients, c)
}

func (wss *WSServer) broadcast(msg []byte) {
	fmt.Println("new message:", string(msg))
	for c := range wss.clients {
		_, err := c.Write(msg)
		if err != nil {
			log.Printf("error writing message to client: %s", c.RemoteAddr().String())
		}
	}
}

func startServer() {
	wss := NewWSServer()
	log.Println("ws server is listening at port :3000...")
	http.Handle("/ws", websocket.Handler(wss.handleClient))
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func startClient() {
	// TODO: skip taken ports (make unique ports)
	origin := fmt.Sprintf("http://localhost:%d/", 3000+rand.Intn(1000))

	url := "ws://localhost:3000/ws"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket server:", err)
	}
	defer ws.Close()
	log.Println("Connected to WebSocket server as client.")

	// Goroutine to read and display incoming messages from the server
	go func() {
		for {
			var msg = make([]byte, 1024)
			n, err := ws.Read(msg)
			if err != nil {
				if err == io.EOF {
					log.Println("Disconnected from server.")
					return
				}
				log.Println("Error reading from server:", err)
				continue
			}
			fmt.Println(string(msg[:n]))
		}
	}()

	// Main loop to read input from stdin and send to the server
	reader := bufio.NewReader(os.Stdin)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Println("Error reading input:", err)
			continue
		}
		msg = strings.TrimSpace(msg)
		if msg == "exit" {
			os.Exit(0)
		}
		if msg == "" {
			continue
		}

		_, err = ws.Write([]byte(msg))
		if err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("    ./broadcast-server server    start the server")
	fmt.Println("    ./broadcast-server client    connect to running server as a client")
}

func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("missing commandline arguments")
		printUsage()
		os.Exit(1)
	}
	switch args[1] {
	case "server":
		startServer()
	case "client":
		startClient()
	default:
		fmt.Printf("invalid argument '%s'\n", args[1])
		printUsage()
		os.Exit(1)
	}
}
