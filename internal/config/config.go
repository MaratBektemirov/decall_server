package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AuthDomain         string
	CORSOrigins        []string
	ChallengeTTL       int
	TurnSecret         string
	TurnHost           string
	TurnRealm          string
	TurnCredentialTTL  int
	TurnTLS            bool
}

func normalizeDomain(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "https://")
	v = strings.TrimPrefix(v, "http://")
	if i := strings.Index(v, "/"); i >= 0 {
		v = v[:i]
	}
	if i := strings.Index(v, ":"); i >= 0 {
		v = v[:i]
	}
	return v
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

	turnTTL := 86400
	if v := strings.TrimSpace(os.Getenv("TURN_CREDENTIAL_TTL_SEC")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			turnTTL = n
		}
	}

	return Config{
		AuthDomain:        normalizeDomain(os.Getenv("AUTH_DOMAIN")),
		CORSOrigins:       origins,
		ChallengeTTL:      ttl,
		TurnSecret:        strings.TrimSpace(os.Getenv("TURN_SECRET")),
		TurnHost:          normalizeDomain(os.Getenv("TURN_HOST")),
		TurnRealm:         normalizeDomain(firstNonEmpty(os.Getenv("TURN_REALM"), os.Getenv("TURN_HOST"), os.Getenv("AUTH_DOMAIN"))),
		TurnCredentialTTL: turnTTL,
		TurnTLS:           envBool("TURN_TLS", true),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func envBool(key string, defaultValue bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultValue
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}
