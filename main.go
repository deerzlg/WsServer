package main

import (
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

var addr = flag.String("addr", ":9099", "http service address")

// roomHubMap是线程安全的map[string]*Hub   记录roomId到*Hub实例的映射
var roomHubMap sync.Map
var roomMutexes = make(map[string]*sync.Mutex)
var mutexForRoomMutexes = new(sync.Mutex)

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()
	r := mux.NewRouter()
	r.HandleFunc("/{room}", serveHome)
	r.HandleFunc("/ws/{room}", func(w http.ResponseWriter, r *http.Request) {
		//提取Path参数
		vars := mux.Vars(r)
		roomId := vars["room"]
		mutexForRoomMutexes.Lock()
		roomMutex, ok := roomMutexes[roomId]
		if ok {
			roomMutex.Lock()
		} else {
			roomMutexes[roomId] = new(sync.Mutex)
			roomMutexes[roomId].Lock()
		}
		mutexForRoomMutexes.Unlock()
		room, ok := roomHubMap.Load(roomId)
		var hub *Hub
		if ok {
			hub = room.(*Hub)
		} else {
			hub = newHub(roomId)
			roomHubMap.Store(roomId, hub)
			go hub.run()
		}
		serveWs(hub, w, r)
	})
	err := http.ListenAndServe(*addr, r)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
