package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/eiannone/keyboard"
)

type Client struct {
	id       string
	conn     net.Conn
	messages chan string
	char     chan rune
}

func main() {
	conn, err := net.Dial("tcp", "localhost:1234")
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}

	defer conn.Close()

	log.Printf("Connected to server %s", conn.RemoteAddr())

	client := Client{
		id:       conn.RemoteAddr().String(),
		conn:     conn,
		messages: make(chan string),
		char:     make(chan rune),
	}

	handleClient(&client)
}

func handleClient(client *Client) {
	defer disconnectClient(client)

	go handleKeyboard(client)

	go buildConsole(client)

	buffer := make([]byte, 1024)

	for {
		n, err := client.conn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from server %s: %s", client.conn.RemoteAddr(), err)

			if err == io.EOF || err == io.ErrClosedPipe || err == io.ErrUnexpectedEOF {
				break
			}
		}

		message := string(buffer[:n])
		client.messages <- message
	}

}

func handleKeyboard(client *Client) {
	err := keyboard.Open()
	if err != nil {
		log.Fatalf("Error opening keyboard: %s", err)
	}

	defer keyboard.Close()

	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			log.Fatalf("Error reading key press: %s", err)
		}

		if key == keyboard.KeyBackspace || key == keyboard.KeyBackspace2 {
			client.char <- 8
		} else if key == keyboard.KeyEnter {
			client.char <- 13
		} else if key == keyboard.KeySpace {
			client.char <- 32
		} else if key == keyboard.KeyEsc {
			log.Fatalf("Program closed by ESC")
		} else {
			client.char <- char
		}
	}
}

func buildConsole(client *Client) {
	serverMessages := make([]string, 10)
	userBuffer := ""

	send := false

	for {
		select {
		case char := <-client.char:
			if char == 8 && len(userBuffer) > 0 {
				userBuffer = userBuffer[:len(userBuffer)-1]
			} else if char == 13 {
				send = true
			} else if len(userBuffer) < 1022 {
				if char == 32 {
					userBuffer += " "
				} else {
					userBuffer += string(char)
				}
			}

			if len(userBuffer) >= 1022 {
				userBuffer = userBuffer[:1022]
			}

			if send && !strings.HasSuffix(userBuffer, "\r\n") {
				userBuffer += "\r\n"
			}
		case message := <-client.messages:
			if len(serverMessages) >= 10 {
				copy(serverMessages[0:], serverMessages[1:])
				serverMessages = serverMessages[:len(serverMessages)-1]
			}

			serverMessages = append(serverMessages, message)
		default:
			continue
		}

		if send {
			_, err := client.conn.Write([]byte(userBuffer))
			if err != nil {
				log.Fatalf("Error writing message to server: %s", err)
			}

			userBuffer = ""
			send = false
		}

		clearConsole()
		for _, message := range serverMessages {
			fmt.Print(message)
		}
		fmt.Print("\n---------------\n")
		fmt.Print("Your message: " + userBuffer)
	}

}

func disconnectClient(client *Client) {
	log.Printf("client %s disconnected", client.id)

	close(client.messages)
	close(client.char)

	err := client.conn.Close()
	if err != nil {
		log.Printf("Error while closing client %s: %s", client.id, err)
	}
}

func clearConsole() {
	switch runtime.GOOS {
	case "linux", "darwin":
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()

	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
