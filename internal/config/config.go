package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AuthDomain  string
	CORSOrigins []string
	ChallengeTTL int
}

func Load() Config {
	ttl := 300
	if v := strings.TrimSpace(os.Getenv("CHALLENGE_TTL_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			ttl = n
		}
	}

	origins := []string{"http://localhost:5173", "http://127.0.0.1:5173"}
	if v := strings.TrimSpace(os.Getenv("CORS_ORIGINS")); v != "" {
		parts := strings.Split(v, ",")
		origins = origins[:0]
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				origins = append(origins, p)
			}
		}
	}

	return Config{
		AuthDomain:   strings.TrimSpace(os.Getenv("AUTH_DOMAIN")),
		CORSOrigins:  origins,
		ChallengeTTL: ttl,
	}
}
