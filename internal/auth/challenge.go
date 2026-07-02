package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"decall_server/internal/config"
	"decall_server/internal/middleware"
)

type Challenge struct {
	Domain string `json:"domain"`
	Nonce  string `json:"nonce"`
	Exp    int64  `json:"exp"`
}

type Handler struct {
	cfg    config.Config
	nonces sync.Map
}

func NewHandler(cfg config.Config) *Handler {
	return &Handler{cfg: cfg}
}

func (h *Handler) IssueChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	domain := h.cfg.AuthDomain
	if domain == "" {
		domain = middleware.RequestHostDomain(r)
	}
	if domain == "" {
		domain = "localhost"
	}

	nonce, err := generateNonce(16)
	if err != nil {
		http.Error(w, "failed to generate nonce", http.StatusInternalServerError)
		return
	}

	exp := time.Now().Unix() + int64(h.cfg.ChallengeTTL)
	h.nonces.Store(nonce, exp)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(Challenge{
		Domain: domain,
		Nonce:  nonce,
		Exp:    exp,
	})
}

func generateNonce(byteLen int) (string, error) {
	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
