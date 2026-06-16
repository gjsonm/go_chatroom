package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

func main() {
	//Inisialisasi mutex
	mtx := sync.Mutex{}

	//Mencoba menghubungkan koneksi ke server di port 9090
	conn, err := net.Dial("tcp", ":9090")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot connect to server!")
		os.Exit(1)
	} else {
		fmt.Fprintf(os.Stderr, "Connected to server!\n")
	}

	connReader := bufio.NewReader(conn)//Variabel yang berfungsi untuk membaca input dari connection(server)
	localReader := bufio.NewReader(os.Stdin) //Variabel yang berfungsi untuk membaca input dari lokal (user 'Terminal')

	mtx.Lock()
	fmt.Print("Your username: ")

	//Meminta user untuk mencantumkan username yang akan digunakan
	name, err := localReader.ReadString('\n')

	if err != nil {
		fmt.Fprintf(os.Stderr, "error name")
	} else {
		fmt.Fprintf(conn, "%s", name)
	}
	mtx.Unlock()

	//Panggil function receivemessage agar kita bisa menerima pesan dari orang lain kapan saja.
	go receiveMessage(connReader)


	//loop utama program client yang bertujuan untuk user mengetik pesan lalu mengirimnya ke server
	for {
		fmt.Print(">>> ")

		//Membaca pesan yang diketik oleh user di terminal (sampai ditekan enter)
		message, _ := localReader.ReadString('\n')
		fmt.Fprintf(conn, "%s", message) // "kirim pesan" ke connection (server)

	}
}

//Function yang berfungsi untuk mengambil atau mendengarkan pesan dari server 
func receiveMessage(connReader *bufio.Reader) {
	//Loop  yang selalu dijalankan yang berfungsi untuk melihat atau memantau apakah ada pesan dari sisi server
	for {
		//Membaca pesan yang masuk dari server
		message, err := connReader.ReadString('\n') // read string from connection

		//mencegah kalo tiba tiba koneksi terputus atau server dimatikan 
		if err != nil {
			fmt.Fprintf(os.Stderr, "bye\n")
			os.Exit(1)
		} else {
			fmt.Printf("\n%s", message)
			fmt.Print(">>> ")
		}
	}
}
