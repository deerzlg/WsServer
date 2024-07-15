package main

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Identity of room.
	roomId string

	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan Message

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub(roomId string) *Hub {
	return &Hub{
		roomId:     roomId,
		broadcast:  make(chan Message),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run() {
	defer func() {
		close(h.unregister)
		close(h.broadcast)
	}()
	for {
		select {
		case client := <-h.unregister:
			roomMutex := roomMutexes[h.roomId]
			roomMutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if len(h.clients) == 0 {
					roomHubMap.Delete(h.roomId)
					roomMutex.Unlock()
					mutexForRoomMutexes.Lock()
					if roomMutex.TryLock() {
						if len(h.clients) == 0 {
							delete(roomMutexes, h.roomId)
						}
						roomMutex.Unlock()
					}
					mutexForRoomMutexes.Unlock()
					return
				}
			}
			roomMutex.Unlock()
		case message := <-h.broadcast:
			roomMutex := roomMutexes[h.roomId]
			roomMutex.Lock()
			for client := range h.clients {
				// Don't send message to sender.
				if client != message.from {
					select {
					case client.send <- message.data:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
			}
			roomMutex.Unlock()
		}
	}
}
