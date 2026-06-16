package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

//Tipe Data yang digunakan untuk menyimpan informasi Nama user dan Nama Ruangan Saat ini
type Client struct {
	Name string
	Room string
}

//Varibel yang menyimpan seluruh client yang lagi aktif di server (Terhubung)
var clients = map[net.Conn]*Client{}

func main() {
	mtx := sync.Mutex{}
	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen")
	} else {
		fmt.Println("Listening port :9090")
	}

	//Loop utama server, Loop terus dijalankan agar bisa atau menunggu koneksi baru (koneksi disini teh koneksi client ke server)
	for {
		//Variabel yang menerima ketika ada koneksi baru
		conn, err := ln.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to accept connection")
		} else {
			fmt.Println("New connection accepted")
		}

		//Memmbuat koneksi baru bisa dijalankan secara pararel
		go newConnection(conn, &mtx)
	}
}

//Function yang mengurus atau menangani User
func newConnection(conn net.Conn, mtx *sync.Mutex) {
	//Variabel buat ngebaca input komunikasi
	reader := bufio.NewReader(conn)

	// Variabel yang membaca input user (Bagian username)
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

	//tolak username kalo udah ada
	if !checkUsername(conn, name, mtx) {
		fmt.Fprintf(conn, "Username already exists\n")
		conn.Close()
		return
	}

	// kasih tau semua orang kalo ada user baru masuk server
	broadcastGlobal(conn, fmt.Sprintf("[SERVER] %s has joined", name), mtx)

	// kasih tau room general kalo ada user baru masuk
	broadcastRoom(conn, "general", fmt.Sprintf("[ROOM general] %s has joined the room", name), mtx)

	// tampilin list command ke client yang baru join
	// "/join Buat masuk room"
	// "/Exit buat keluar dari room dan kembali ke general"
	// "/Rooms buat ada room apa aja yang aktif (tersedia)"
	// "/Help buat ngeliatin ada command apa aja dan gunannya buat apa"
	fmt.Fprintf(conn, "Untuk masuk ke room: /join <namaroom>\n")
	fmt.Fprintf(conn, "Untuk balik ke general: /exit\n")
	fmt.Fprintf(conn, "Untuk lihat room aktif: /rooms\n")
	fmt.Fprintf(conn, "Untuk lihat list command: /help\n")

	//Loop yang digunakan untuk membaca setiap input (ketikan) yang dilakukan user 
	for {
		message, err := reader.ReadString('\n')
		//if ini kalo tb tb kalian client/user terputus dari server 
		if err != nil {
			break
		}

		message = strings.TrimSpace(message) //bersihkan space di awal dan akhir kalimat

		if strings.HasPrefix(message, "/join") { //jika message yang dimasukkan berupa perintah buat join room
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
			
		} else if message == "/exit" { //jika message yang dimasukkan berupa perintah keluar dari room saat ini
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

		} else if message == "/rooms" { //Jika message yang dimasukan berupa perintah "Melihat room yang aktif"
			// tampilin daftar room yang aktif
			listRooms(conn, mtx)
			continue

		} else if message == "/help" {//Jika message yang dimasukan berupa perintah yang menunjukan list command yg tersedia
			// tampilin list command
			fmt.Fprintf(conn, "Untuk masuk ke room: /join <namaroom>\n")
			fmt.Fprintf(conn, "Untuk balik ke general: /exit\n")
			fmt.Fprintf(conn, "Untuk lihat room aktif: /rooms\n")
			fmt.Fprintf(conn, "Untuk lihat list command: /help\n")
			continue
		}

		//Kalo perintah yang diberikan selain ke 4 itu, nanti bakal dikasih tau 
		broadcastMessage(conn, message, mtx)
	}

	// kasih tau semua orang kalo user cabut dari server
	mtx.Lock()
	userName := clients[conn].Name
	mtx.Unlock()
	broadcastGlobal(conn, fmt.Sprintf("[SERVER] %s has left", userName), mtx)

	mtx.Lock()
	delete(clients, conn) // Hapus Client atau user dari daftar user aktif (map)
	mtx.Unlock()
	conn.Close() // Tutup atau hapus koneksi user
}

// function untuk broadcasting pesan chat dari suatu user ke semua user yang ada didalam room
func broadcastMessage(conn net.Conn, message string, mtx *sync.Mutex) {
	mtx.Lock()
	roomPengirim := clients[conn].Room

	//Loop buat ngelakuin iterasi sebanyak user aktif yang yang ada di room tersebut
	for client, peopleInRoom := range clients {
		if client == conn { // wont send message to itself
			continue
		}

		//Kalo user tersebut ada didalam room yang sama dengan pengirim broadcast, kirim pesannya
		if peopleInRoom.Room == roomPengirim {
			fmt.Fprintf(client, "%s\t: %s\n", clients[conn].Name, message)
		}
	}
	mtx.Unlock()
}

// Function yang mengurus broadcast ke semua client kalo ada client yang join atau leave (join/leave server)
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

// Function yang mengurus broadcast ke client di room tertentu aja (join/leave room)
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

// Function yang mengurus tampilin daftar room yang lagi aktif (minimal 1 orang di dalamnya)
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

//Function yang mengurus validasi dari username yang akan digunakan user. Validasi berupa (sudah dipakai user lain atau belum)
func checkUsername(conn net.Conn, username string, mtx *sync.Mutex) bool {
	mtx.Lock()
	for _, user := range clients {
		//Kondisi dimana jika terdapat nama yang sama
		if username == user.Name {
			mtx.Unlock()
			return false
		}
	}
	// Masukin user baru ke map of clients dengan room awal yang dimasukin general (defaultnya)
	clients[conn] = &Client{
		Name: username,
		Room: "general",
	}

	mtx.Unlock()
	return true //Berita tahu bahwa username telah berhasil didaftarkan
}
