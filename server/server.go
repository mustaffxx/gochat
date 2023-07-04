package main

import (
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	id       string
	conn     net.Conn
	messages chan string
}

func main() {
	var clientMap sync.Map

	listener, err := net.Listen("tcp", "localhost:1234")
	if err != nil {
		log.Fatalf("Server start error: %s", err)
	}
	defer listener.Close()

	log.Printf("Server started on %s", listener.Addr())

	for {
		c, err := listener.Accept()
		if err != nil {
			log.Printf("Connect acception error: %s", err)
			continue
		}

		uid := generateUniqueName(&clientMap)
		client := Client{
			id:       uid,
			conn:     c,
			messages: make(chan string),
		}

		log.Printf("client %s connected", client.id)

		clientMap.Store(client.id, client)

		go handleClient(&client, &clientMap)

	}
}

func handleClient(client *Client, clientMap *sync.Map) {
	defer disconnectClient(client, clientMap)

	go dispatchMessage(client)

	buffer := make([]byte, 1024)

	for {
		n, err := client.conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from client %s: %s", client.id, err)

			if err == io.EOF || err == io.ErrClosedPipe || err == io.ErrUnexpectedEOF {
				break
			}
		}

		message := string(buffer[:n])
		log.Printf("client [%s]: %s", client.id, message)

		clientMap.Range(func(key, value interface{}) bool {
			otherClient := value.(Client)
			otherClient.messages <- message

			return true
		})
	}
}

func dispatchMessage(client *Client) {
	for message := range client.messages {
		timestamp := "[" + time.Now().Format("02/01/2006 15:04:05") + "]"
		message = timestamp + " " + message
		_, err := client.conn.Write([]byte(message))
		if err != nil {
			log.Printf("Error sending message to client %s: %s", client.id, err)

			if err == io.EOF || err == io.ErrClosedPipe {
				break
			}
		}
	}
}

func disconnectClient(client *Client, clientMap *sync.Map) {
	log.Printf("client %s disconnected", client.id)

	close(client.messages)

	clientMap.Delete(client.id)

	err := client.conn.Close()
	if err != nil {
		log.Printf("Error while closing client %s: %s", client.id, err)
	}
}

func generateUniqueName(clientMap *sync.Map) string {
	var uid string

	for {
		uid = uuid.New().String()[:8]

		_, exists := clientMap.Load(uid)

		if !exists {
			break
		}
	}

	return uid
}
