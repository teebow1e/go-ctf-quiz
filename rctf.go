package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

const rctfRequestTimeout = 10 * time.Second

// verifyRCTF exchanges a team token for an auth token, then looks up the team
// it belongs to. Team tokens never expire and grant full account access, so
// neither they nor the auth token may be logged.
func verifyRCTF(baseURL, input string) (Identity, error) {
	teamToken := extractTeamToken(input)
	if teamToken == "" {
		return Identity{}, errors.New("empty team token")
	}

	payload, err := json.Marshal(map[string]string{"teamToken": teamToken})
	if err != nil {
		return Identity{}, fmt.Errorf("failed to encode login request: %w", err)
	}

	status, body, err := rctfRequest(fasthttp.MethodPost, baseURL+"/api/v1/auth/login", "", payload)
	if err != nil {
		log.Printf("[rctf-login] request failed: %v\n", err)
		return Identity{}, errAuthUnavailable
	}

	kind := gjson.Get(body, "kind").String()
	authToken := gjson.Get(body, "data.authToken").String()
	switch {
	case status == 200 && kind == "goodLogin" && authToken != "":
	case kind == "badTokenVerification" || kind == "badUnknownUser" || kind == "badBody":
		log.Printf("[rctf-login] token rejected: %s\n", kind)
		return Identity{}, errors.New("team token rejected")
	default:
		log.Printf("[rctf-login] unexpected response: status=%d kind=%q\n", status, kind)
		return Identity{}, errAuthUnavailable
	}

	status, body, err = rctfRequest(fasthttp.MethodGet, baseURL+"/api/v1/users/me", authToken, nil)
	if err != nil {
		log.Printf("[rctf-profile] request failed: %v\n", err)
		return Identity{}, errAuthUnavailable
	}

	kind = gjson.Get(body, "kind").String()
	id := gjson.Get(body, "data.id").String()
	name := gjson.Get(body, "data.name").String()
	if status != 200 || kind != "goodUserSelfData" || id == "" || name == "" {
		log.Printf("[rctf-profile] unexpected response: status=%d kind=%q\n", status, kind)
		return Identity{}, errAuthUnavailable
	}

	log.Printf("[rctf] authenticated team %q (%s)\n", name, id)
	return Identity{Name: name, LogKey: id}, nil
}

// extractTeamToken accepts a raw team token or an invite link of the form
// https://ctf.example.com/login?token=<team token>.
func extractTeamToken(input string) string {
	if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
		return input
	}
	parsed, err := url.Parse(input)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("token")
}

func rctfRequest(method, requestURL, bearerToken string, body []byte) (int, string, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(requestURL)
	req.Header.SetMethod(method)
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	if body != nil {
		req.Header.SetContentType("application/json")
		req.SetBody(body)
	}

	if err := fasthttp.DoTimeout(req, resp, rctfRequestTimeout); err != nil {
		return 0, "", err
	}
	return resp.StatusCode(), string(resp.Body()), nil
}
