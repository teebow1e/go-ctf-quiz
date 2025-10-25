package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

func verifyIdentity(token string) (string, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(os.Getenv("CTFD_URL") + "/api/v1/users/me")
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	if err := fasthttp.Do(req, resp); err != nil {
		log.Printf("[call-api-CTFd] error while pinging CTFd: %v\n", err)
		return "", fmt.Errorf("failed to connect to CTFd: %w", err)
	}

	bodyStr := string(resp.Body())

	if resp.StatusCode() == 200 {
		user := gjson.Get(bodyStr, "data.name")
		email := gjson.Get(bodyStr, "data.email")

		log.Printf("[%s] Token OK\n", token)
		log.Printf("[%s] Got identity: %s - %s", token, user.Str, email.Str)
		return user.Str, nil
	} else {
		log.Printf("[%s] Token NOT OK\n", token)
		log.Printf("[%s] Received error: %s", token, bodyStr)
		return "", errors.New("failed to validate identity")
	}
}
