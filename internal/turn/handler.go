package turn

import (
	"encoding/json"
	"net/http"

	"decall_server/internal/auth"
	"decall_server/internal/config"
)

type Handler struct {
	cfg  config.Config
	auth *auth.Handler
}

func NewHandler(cfg config.Config, authHandler *auth.Handler) *Handler {
	return &Handler{cfg: cfg, auth: authHandler}
}

type credentialsRequest struct {
	Proof auth.SecretAuthProof `json:"proof"`
}

func (h *Handler) IssueCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if err := h.auth.VerifyProof(h.cfg, req.Proof); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	pubKeyID := PubKeyID(req.Proof.PubKey.Value)
	resp, err := BuildCredentials(h.cfg, pubKeyID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
