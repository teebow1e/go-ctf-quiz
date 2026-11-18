package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// errAuthUnavailable means the token could not be checked, as opposed to the
// token being rejected.
var errAuthUnavailable = errors.New("authentication service unavailable")

// Identity is an authenticated player.
type Identity struct {
	Name string
	// LogKey identifies the player in attempt logs and retry counts.
	LogKey string
}

// Authenticator verifies the token a player enters when connecting.
type Authenticator struct {
	TokenPrompt string
	Verify      func(token string) (Identity, error)
}

// newAuthenticator builds the authenticator selected by the environment. It
// returns nil when STRICT_MODE is off, which lets players in as guests.
func newAuthenticator() (*Authenticator, error) {
	if os.Getenv("STRICT_MODE") != "1" {
		return nil, nil
	}

	limited := func(verify func(string) (Identity, error)) func(string) (Identity, error) {
		return newVerifyLimiter(verify, envInt("AUTH_RATE_PER_SECOND", 10)).Verify
	}

	switch provider := os.Getenv("AUTH_PROVIDER"); provider {
	case "", "ctfd":
		return &Authenticator{
			TokenPrompt: "Please enter your CTFd access token: ",
			Verify: limited(func(token string) (Identity, error) {
				name, err := verifyIdentity(token)
				if err != nil {
					return Identity{}, err
				}
				return Identity{Name: name, LogKey: token}, nil
			}),
		}, nil
	case "rctf":
		baseURL := strings.TrimRight(os.Getenv("RCTF_URL"), "/")
		parsed, err := url.Parse(baseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return nil, fmt.Errorf("RCTF_URL must be an http(s) URL, got %q", baseURL)
		}
		return &Authenticator{
			TokenPrompt: "Please enter your rCTF team token or invite link: ",
			Verify: limited(func(token string) (Identity, error) {
				return verifyRCTF(baseURL, token)
			}),
		}, nil
	default:
		return nil, fmt.Errorf("unknown AUTH_PROVIDER %q (expected ctfd or rctf)", provider)
	}
}
