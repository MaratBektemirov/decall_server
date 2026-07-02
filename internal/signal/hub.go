package signal

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
)

const maxPeersPerRoom = 2

type Hub struct {
	rooms sync.Map
}

type room struct {
	id    string
	peers map[*peer]struct{}
	mu    sync.Mutex
}

type peer struct {
	hub    *Hub
	room   *room
	role   string
	send   chan []byte
	closed bool
}

func NewHub() *Hub {
	return &Hub{}
}

type Inbound struct {
	Type      string          `json:"type"`
	RoomID    string          `json:"roomId"`
	Role      string          `json:"role"`
	SDP       string          `json:"sdp"`
	Candidate json.RawMessage `json:"candidate"`
}

type Outbound struct {
	Type    string          `json:"type"`
	Role    string          `json:"role,omitempty"`
	SDP     string          `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Message string          `json:"message,omitempty"`
}

func (h *Hub) Join(roomID string, role string) (*peer, error) {
	if roomID == "" {
		return nil, errors.New("roomId required")
	}

	r := h.getOrCreateRoom(roomID)

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.peers) >= maxPeersPerRoom {
		return nil, errors.New("room is full")
	}

	if role == "" {
		if len(r.peers) == 0 {
			role = "host"
		} else {
			role = "guest"
		}
	}

	p := &peer{
		hub:  h,
		room: r,
		role: role,
		send: make(chan []byte, 32),
	}
	r.peers[p] = struct{}{}

	if len(r.peers) == maxPeersPerRoom {
		out := Outbound{Type: "peer-joined"}
		for peer := range r.peers {
			peer.write(out)
		}
	} else {
		p.write(Outbound{Type: "waiting"})
	}

	return p, nil
}

func (h *Hub) getOrCreateRoom(id string) *room {
	if v, ok := h.rooms.Load(id); ok {
		return v.(*room)
	}

	r := &room{
		id:    id,
		peers: make(map[*peer]struct{}),
	}
	actual, _ := h.rooms.LoadOrStore(id, r)
	return actual.(*room)
}

func (p *peer) Handle(msg Inbound) {
	switch msg.Type {
	case "offer", "answer", "ice":
		p.relay(Outbound{
			Type:      msg.Type,
			SDP:       msg.SDP,
			Candidate: msg.Candidate,
		})
	default:
		p.write(Outbound{Type: "error", Message: "unknown message type"})
	}
}

func (p *peer) relay(out Outbound) {
	p.room.mu.Lock()
	defer p.room.mu.Unlock()

	for other := range p.room.peers {
		if other == p {
			continue
		}
		other.write(out)
	}
}

func (p *peer) write(out Outbound) {
	data, err := json.Marshal(out)
	if err != nil {
		return
	}

	select {
	case p.send <- data:
	default:
		log.Printf("signal: peer send buffer full, dropping")
	}
}

func (p *peer) Leave() {
	if p.closed {
		return
	}
	p.closed = true
	close(p.send)

	p.room.mu.Lock()
	delete(p.room.peers, p)
	empty := len(p.room.peers) == 0
	remaining := make([]*peer, 0, len(p.room.peers))
	for other := range p.room.peers {
		remaining = append(remaining, other)
	}
	p.room.mu.Unlock()

	for _, other := range remaining {
		other.write(Outbound{Type: "peer-left"})
	}

	if empty {
		p.hub.rooms.Delete(p.room.id)
	}
}
