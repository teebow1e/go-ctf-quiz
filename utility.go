package main

import (
	"errors"
	"fmt"
	"log"
	"unsafe"

	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

func B2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func verifyIdentity(token string) (string, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI("http://192.168.194.130:8000" + "/api/v1/users/me")
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	if err := fasthttp.Do(req, resp); err != nil {
		log.Fatalf("error while pinging CTFd: %v\n", err)
	}

	if resp.StatusCode() == 200 {
		user := gjson.Get(B2S(resp.Body()), "data.name")
		email := gjson.Get(B2S(resp.Body()), "data.email")

		log.Printf("[%s] Token OK\n", token)
		log.Printf("[%s] Got identity: %s - %s", token, user.Str, email.Str)
		return user.Str, nil
	} else {
		log.Printf("[%s] Token NOT OK\n", token)
		log.Printf("[%s] Received error: %s", token, string(resp.Body()))
		return "", errors.New("failed to validate identity")
	}
}
