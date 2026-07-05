package turn

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"decall_server/internal/config"
)

type IceServer struct {
	URLs       any    `json:"urls"`
	Username   string `json:"username,omitempty"`
	Credential string `json:"credential,omitempty"`
}

type CredentialsResponse struct {
	IceServers []IceServer `json:"iceServers"`
}

func BuildCredentials(cfg config.Config, pubKeyID string) (CredentialsResponse, error) {
	if cfg.TurnSecret == "" {
		return CredentialsResponse{}, fmt.Errorf("turn is not configured")
	}
	if cfg.TurnHost == "" {
		return CredentialsResponse{}, fmt.Errorf("turn host is not configured")
	}

	ttl := time.Duration(cfg.TurnCredentialTTL) * time.Second
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	expiry := time.Now().Add(ttl).Unix()
	username := fmt.Sprintf("%d:%s", expiry, pubKeyID)
	mac := hmac.New(sha1.New, []byte(cfg.TurnSecret))
	_, _ = mac.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	host := cfg.TurnHost
	realm := cfg.TurnRealm
	if realm == "" {
		realm = host
	}
	_ = realm

	urls := []string{
		fmt.Sprintf("stun:%s:3478", host),
		fmt.Sprintf("turn:%s:3478?transport=udp", host),
		fmt.Sprintf("turn:%s:3478?transport=tcp", host),
	}
	if cfg.TurnTLS {
		urls = append(urls, fmt.Sprintf("turns:%s:5349?transport=tcp", host))
	}

	return CredentialsResponse{
		IceServers: []IceServer{
			{URLs: urls[0]},
			{
				URLs:       urls[1:],
				Username:   username,
				Credential: credential,
			},
		},
	}, nil
}

func PubKeyID(pubKeyValue string) string {
	sum := sha256.Sum256([]byte(pubKeyValue))
	return hex.EncodeToString(sum[:8])
}
