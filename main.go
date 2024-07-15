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

// 对于每个room，都有一个锁，小锁实现细粒度的控制
var roomMutexes = make(map[string]*sync.Mutex)

// 对于roomMutexes的锁
var mutexForRoomMutexes = new(sync.Mutex)

func main() {
	flag.Parse()
	r := mux.NewRouter()
	r.HandleFunc("/ws/{room}", func(w http.ResponseWriter, r *http.Request) {
		//提取Path参数
		vars := mux.Vars(r)
		roomId := vars["room"]
		mutexForRoomMutexes.Lock()
		//先取房间锁，如果不存在则创建
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
