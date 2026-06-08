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

	// tolak kalau username kosong
	if name == "" {
		fmt.Fprintf(conn, "Username tidak boleh kosong\n")
		conn.Close()
		return
	}

	if !checkUsername(conn, name, mtx) {
		fmt.Fprintf(conn, "Username already exists\n")
		conn.Close()
		return
	}

	// kasih tau semua orang kalo ada user baru masuk server
	broadcastGlobal(conn, fmt.Sprintf("[SERVER] %s has joined", name), mtx)
	// kasih tau room general kalo user masuk
	broadcastRoom(conn, "general", fmt.Sprintf("[ROOM general] %s has joined the room", name), mtx)

	// tampilin list command ke client yang baru join
	fmt.Fprintf(conn, "Untuk masuk ke room: /join <namaroom>\n")
	fmt.Fprintf(conn, "Untuk balik ke general: /exit\n")
	fmt.Fprintf(conn, "Untuk lihat room aktif: /rooms\n")
	fmt.Fprintf(conn, "Untuk lihat list command: /help\n")

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		message = strings.TrimSpace(message) //bersihkan space di awal dan akhir kalimat

		if strings.HasPrefix(message, "/join") { //jika message yang dimasukkan berawalan "/join"
			// ambil nama room dari command
			room := strings.TrimSpace(strings.TrimPrefix(message, "/join"))

			// tolak kalo room kosong
			if room == "" {
				fmt.Fprintf(conn, "Nama room tidak boleh kosong. Gunakan: /join <namaroom>\n")
				continue
			}

			mtx.Lock()
			oldRoom := clients[conn].Room
			clients[conn].Room = room //masukkan client ke room sesuai dengan room yg diketik
			mtx.Unlock()

			// kasih tau room lama kalo user pergi
			broadcastRoom(conn, oldRoom, fmt.Sprintf("[ROOM %s] %s has left the room", oldRoom, name), mtx)
			// kasih tau room baru kalo user masuk
			broadcastRoom(conn, room, fmt.Sprintf("[ROOM %s] %s has joined the room", room, name), mtx)

			fmt.Fprintf(conn, "Kamu berhasil masuk ke room: %s\n", room)
			continue
		} else if message == "/exit" { //jika message yang dimasukkan adalah "/exit"
			mtx.Lock()
			oldRoom := clients[conn].Room
			clients[conn].Room = "general" //kembalikan client ke room general (room utama)
			mtx.Unlock()

			// kasih tau room lama kalo user keluar
			broadcastRoom(conn, oldRoom, fmt.Sprintf("[ROOM %s] %s has left the room", oldRoom, name), mtx)
			// kasih tau general kalo user balik
			broadcastRoom(conn, "general", fmt.Sprintf("[ROOM general] %s has joined the room", name), mtx)

			fmt.Fprintf(conn, "Kamu berhasil kembali ke room utama\n")
			continue
		} else if message == "/rooms" {
			// tampilin daftar room yang aktif
			listRooms(conn, mtx)
			continue
		} else if message == "/help" {
			// tampilin list command
			fmt.Fprintf(conn, "Untuk masuk ke room: /join <namaroom>\n")
			fmt.Fprintf(conn, "Untuk balik ke general: /exit\n")
			fmt.Fprintf(conn, "Untuk lihat room aktif: /rooms\n")
			fmt.Fprintf(conn, "Untuk lihat list command: /help\n")
			continue
		}

		broadcastMessage(conn, message, mtx)
	}

	// kasih tau semua orang kalo user cabut dari server
	mtx.Lock()
	userName := clients[conn].Name
	mtx.Unlock()
	broadcastGlobal(conn, fmt.Sprintf("[SERVER] %s has left", userName), mtx)

	mtx.Lock()
	delete(clients, conn) // remove client from client list
	mtx.Unlock()
	conn.Close() // close this clients connection
}

// function for broadcasting a clients message to another clients in the same room
func broadcastMessage(conn net.Conn, message string, mtx *sync.Mutex) {
	mtx.Lock()
	roomPengirim := clients[conn].Room

	for client, peopleInRoom := range clients {
		if client == conn { // wont send message to itself
			continue
		}

		if peopleInRoom.Room == roomPengirim {
			fmt.Fprintf(client, "%s\t: %s\n", clients[conn].Name, message)
		}
	}
	mtx.Unlock()
}

// broadcast ke semua client (join/leave server)
func broadcastGlobal(conn net.Conn, msg string, mtx *sync.Mutex) {
	mtx.Lock()
	for client := range clients {
		if client == conn {
			continue
		}
		fmt.Fprintf(client, "%s\n", msg)
	}
	mtx.Unlock()
}

// broadcast ke client di room tertentu aja (join/leave room)
func broadcastRoom(sender net.Conn, room string, msg string, mtx *sync.Mutex) {
	mtx.Lock()
	for client, info := range clients {
		if client == sender {
			continue
		}
		if info.Room == room {
			fmt.Fprintf(client, "%s\n", msg)
		}
	}
	mtx.Unlock()
}

// tampilin daftar room yang lagi aktif (minimal 1 orang di dalamnya)
func listRooms(conn net.Conn, mtx *sync.Mutex) {
	mtx.Lock()
	roomSet := map[string]int{}
	for _, info := range clients {
		roomSet[info.Room]++
	}
	mtx.Unlock()

	fmt.Fprintf(conn, "Room aktif:\n")
	for room, count := range roomSet {
		fmt.Fprintf(conn, "- %s (%d orang)\n", room, count)
	}
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
	clients[conn] = &Client{
		Name: username,
		Room: "general",
	}

	mtx.Unlock()
	return true
}
