package main

import (
	"io"
	"log"
	"net"
	"sync"
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

		client := Client{
			id:       c.LocalAddr().String(),
			conn:     c,
			messages: make(chan string),
		}

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
			otherClient := value.(*Client)

			if client.id != otherClient.id {
				otherClient.messages <- message
			}

			return true
		})
	}
}

func dispatchMessage(client *Client) {
	for message := range client.messages {
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
	close(client.messages)

	clientMap.Delete(client.id)

	err := client.conn.Close()
	if err != nil {
		log.Printf("Error while closing client %s: %s", client.id, err)
	}
}
