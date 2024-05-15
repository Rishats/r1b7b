package main

import (
	"embed"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed static/index.html
var indexHTML embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	connections []*websocket.Conn
	connMutex   sync.Mutex
)

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := indexHTML.ReadFile("static/index.html")
		w.Write(data)
	})

	go udpListener()

	log.Fatal(http.ListenAndServe(":5000", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	connMutex.Lock()
	connections = append(connections, conn)
	connMutex.Unlock()
}

func udpListener() {
	addr := net.UDPAddr{
		Port: 5005,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		data := string(buffer[:n])
		connMutex.Lock()
		for i := 0; i < len(connections); i++ {
			err := connections[i].WriteMessage(websocket.TextMessage, []byte(data))
			if err != nil {
				log.Println("Write error:", err)
				connections[i].Close()
				connections = append(connections[:i], connections[i+1:]...)
				i-- // Adjust index after removing an element
			}
		}
		connMutex.Unlock()
	}
}
