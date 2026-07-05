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
	id       string
	peers    map[*peer]struct{}
	approved bool
	mu       sync.Mutex
}

type peer struct {
	hub    *Hub
	room   *room
	role   string
	callID string
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
	CallID    string          `json:"callId,omitempty"`
	SDP       string          `json:"sdp"`
	Candidate json.RawMessage `json:"candidate"`
}

type Outbound struct {
	Type      string          `json:"type"`
	Role      string          `json:"role,omitempty"`
	CallID    string          `json:"callId,omitempty"`
	SDP       string          `json:"sdp,omitempty"`
	Candidate json.RawMessage `json:"candidate,omitempty"`
	Message   string          `json:"message,omitempty"`
}

func (h *Hub) Join(roomID, role, callID string) (*peer, error) {
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
		hub:    h,
		room:   r,
		role:   role,
		callID: callID,
		send:   make(chan []byte, 32),
	}
	r.peers[p] = struct{}{}

	if len(r.peers) == maxPeersPerRoom {
		host, guest := roomHostGuest(r.peers)
		if host != nil && guest != nil {
			r.approved = false
			host.write(Outbound{Type: "join-request", CallID: guest.callID})
			guest.write(Outbound{Type: "waiting-approval"})
			return p, nil
		}

		r.approved = true
		out := Outbound{Type: "peer-joined"}
		for peer := range r.peers {
			peer.write(out)
		}
		return p, nil
	}

	p.write(Outbound{Type: "waiting"})
	return p, nil
}

func roomHostGuest(peers map[*peer]struct{}) (*peer, *peer) {
	var host, guest *peer
	for p := range peers {
		switch p.role {
		case "host":
			host = p
		case "guest":
			guest = p
		}
	}
	return host, guest
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
	case "accept-guest":
		if err := p.acceptGuest(); err != nil {
			p.write(Outbound{Type: "error", Message: err.Error()})
		}
	case "reject-guest":
		if err := p.rejectGuest(); err != nil {
			p.write(Outbound{Type: "error", Message: err.Error()})
		}
	case "offer", "answer", "ice":
		p.room.mu.Lock()
		approved := p.room.approved
		p.room.mu.Unlock()
		if !approved {
			p.write(Outbound{Type: "error", Message: "join not approved yet"})
			return
		}
		p.relay(Outbound{
			Type:      msg.Type,
			SDP:       msg.SDP,
			Candidate: msg.Candidate,
		})
	default:
		p.write(Outbound{Type: "error", Message: "unknown message type"})
	}
}

func (p *peer) acceptGuest() error {
	if p.role != "host" {
		return errors.New("only host can accept guests")
	}

	p.room.mu.Lock()
	defer p.room.mu.Unlock()

	if len(p.room.peers) != maxPeersPerRoom {
		return errors.New("no pending guest")
	}
	if p.room.approved {
		return errors.New("guest already accepted")
	}

	host, guest := roomHostGuest(p.room.peers)
	if host == nil || guest == nil {
		return errors.New("invalid room state")
	}

	p.room.approved = true
	out := Outbound{Type: "peer-joined"}
	for peer := range p.room.peers {
		peer.write(out)
	}
	return nil
}

func (p *peer) rejectGuest() error {
	if p.role != "host" {
		return errors.New("only host can reject guests")
	}

	p.room.mu.Lock()
	defer p.room.mu.Unlock()

	if len(p.room.peers) != maxPeersPerRoom {
		return errors.New("no pending guest")
	}

	_, guest := roomHostGuest(p.room.peers)
	if guest == nil {
		return errors.New("no guest to reject")
	}

	guest.write(Outbound{Type: "join-rejected", Message: "Host declined your request"})
	p.removeFromRoom(guest)
	p.room.approved = false
	p.write(Outbound{Type: "waiting"})
	return nil
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

func (p *peer) removeFromRoom(target *peer) {
	if target.room != p.room {
		return
	}

	target.closed = true
	close(target.send)
	delete(p.room.peers, target)
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
	p.room.approved = false
	remaining := make([]*peer, 0, len(p.room.peers))
	for other := range p.room.peers {
		remaining = append(remaining, other)
	}
	roomID := p.room.id
	p.room.mu.Unlock()

	for _, other := range remaining {
		other.write(Outbound{Type: "peer-left", Message: "Other participant disconnected"})
	}

	if empty {
		p.hub.rooms.Delete(roomID)
	}
}
