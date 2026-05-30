package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

func main() {
	mtx := sync.Mutex{}
	conn, err := net.Dial("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to server!")
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "Connected to server!\n")
	}

	connReader := bufio.NewReader(conn)      // to read input from connection
	localReader := bufio.NewReader(os.Stdin) // to read input from local

	mtx.Lock()
	fmt.Print("Your username: ")
	name, err := localReader.ReadString('\n')

	if err != nil {
		fmt.Fprintf(os.Stderr, "error name")
	} else {
		fmt.Fprintf(conn, "%s", name)
	}
	mtx.Unlock()

	go receiveMessage(connReader)

	for {
		fmt.Print(">>> ")
		message, _ := localReader.ReadString('\n') // read string input

		fmt.Fprintf(conn, "%s", message) // "kirim pesan" ke connection (server)

	}
}

func receiveMessage(connReader *bufio.Reader) {
	for {
		message, err := connReader.ReadString('\n') // read string from connection

		if err != nil {
			fmt.Fprintf(os.Stderr, "bye\n")
			os.Exit(1)
		} else {
			fmt.Printf("\n%s", message)
			fmt.Print(">>> ")
		}
	}
}
