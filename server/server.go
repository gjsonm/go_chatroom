package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type Client struct {
	Name string
	Room string
}
var clients = map[net.Conn]*Client{}

func main() {
	mtx := sync.Mutex{}
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen")
	} else {
		fmt.Println("Listening port :9090")
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection")
		} else {
			fmt.Println("New connection accepted")
		}

		go newConnection(conn, &mtx)
	}
}

func newConnection(conn net.Conn, mtx *sync.Mutex) {
	reader := bufio.NewReader(conn)

	// detect the clients identity and add it to map
	name, err := reader.ReadString('\n')

	if err != nil {
		return
	} else {
		name = strings.TrimSpace(name)
	}

	if !checkUsername(conn, name, mtx) {
		fmt.Fprintf(conn, "Username already exists\n")
		conn.Close()
		return
	}

	broadcastLog(conn, "has joined", mtx)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		message = strings.TrimSpace(message) //bersihkan space di awal dan akhir kalimat
		if strings.HasPrefix(message, "/join "){ //jika message yang dimasukkan berawalan "/join"
			room := strings.TrimPrefix(message, "/join")
			mtx.Lock()
			clients[conn].Room = room //masukkan client ke room sesuai dengan room yg diketik
			mtx.Unlock()
			fmt.Fprintf(conn, "Kamu berhasil masuk ke room: %s\n", room)
			continue
		} else if message == "/exit"{ //jika message yang dimasukkan adalah "/exit"
			mtx.Lock()
			clients[conn].Room = "general" //kembalikan client ke room general (room utama)
			mtx.Unlock()
			fmt.Fprintf(conn, "Kamu berhasil kembali ke room utama\n")
			continue
		}
		broadcastMessage(conn, message, mtx)
	}

	broadcastLog(conn, "has left", mtx)
	mtx.Lock()
	delete(clients, conn) // remove client from client list
	mtx.Unlock()
	conn.Close() // close this clients connection
}

// function for broadcasting a clients message to another clients
func broadcastMessage(conn net.Conn, message string, mtx *sync.Mutex) {
	mtx.Lock()
	roomPengirim := clients[conn].Room

	for client, peopleInRoom := range clients {
		if client == conn { // wont send message to itself
			continue
		} 
		
		if peopleInRoom.Room == roomPengirim {
			fmt.Fprintf(client, "%s	: %s\n", clients[conn].Name, message)
		}
	}
	mtx.Unlock()
}

// function for broadcasting system log (client disconnected)
func broadcastLog(conn net.Conn, log string, mtx *sync.Mutex) {
	mtx.Lock()
	for client := range clients {
		if client == conn { // wont send message to itself
			continue
		} else {
			fmt.Fprintf(client, "%s %s\n", clients[conn].Name, log)
		}
	}
	mtx.Unlock()
}

func checkUsername(conn net.Conn, username string, mtx *sync.Mutex) bool {
	mtx.Lock()
	for _, user := range clients {
		if username == user.Name {
			mtx.Unlock()
			return false
		}
	}
	// use mutex so that when 2 clients connected, the clients wont be accessed in the same time
	// clients[conn] = username
	clients[conn] = &Client{
		Name: username,
		Room: "general", 
	}

	mtx.Unlock()
	return true
}
