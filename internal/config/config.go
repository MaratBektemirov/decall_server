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

	origins := []string{}
	if v := strings.TrimSpace(os.Getenv("CORS_ORIGINS")); v != "" {
		for _, p := range strings.Split(v, ",") {
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
