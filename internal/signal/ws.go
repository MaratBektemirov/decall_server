package signal

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	"decall_server/internal/config"
	"decall_server/internal/words"
)

type Handler struct {
	hub      *Hub
	upgrader websocket.Upgrader
}

func NewHandler(cfg config.Config, hub *Hub) *Handler {
	allowed := make(map[string]struct{}, len(cfg.CORSOrigins))
	for _, origin := range cfg.CORSOrigins {
		allowed[origin] = struct{}{}
	}

	return &Handler{
		hub: hub,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("signal: upgrade: %v", err)
		return
	}

	var peer *peer

	defer func() {
		if peer != nil {
			peer.Leave()
		}
		conn.Close()
	}()

	for {
		var msg Inbound
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		if msg.Type == "join" {
			if peer != nil {
				peer.write(Outbound{Type: "error", Message: "already joined"})
				continue
			}

			roomID, err := words.NormalizeCallID(msg.RoomID)
			if err != nil {
				_ = conn.WriteJSON(Outbound{Type: "error", Message: err.Error()})
				continue
			}

			joined, err := h.hub.Join(roomID, msg.Role, msg.CallID)
			if err != nil {
				_ = conn.WriteJSON(Outbound{Type: "error", Message: err.Error()})
				continue
			}

			peer = joined
			joined.write(Outbound{Type: "joined", Role: joined.role})

			go h.writePump(conn, joined)
			continue
		}

		if peer == nil {
			_ = conn.WriteJSON(Outbound{Type: "error", Message: "join first"})
			continue
		}

		peer.Handle(msg)
	}
}

func (h *Handler) writePump(conn *websocket.Conn, p *peer) {
	for data := range p.send {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}
