package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type CallAsyncLoopAppend struct {
}

func (callAsync *CallAsyncLoopAppend) JsonPostRequest(client *http.Client, url string, request JSONValue) *HttpResult {
	encoded, err := json.Marshal(request)
	if err != nil {
		log.Fatalf("[FATAL] Failed to encode JSON request: %v", err)
	}
	log.Printf("[INFO] HTTP Post to url: %s", url)
	resp, err := client.Post(url, "application/json", bytes.NewReader(encoded))
	if err != nil {
		log.Printf("[ERROR] HTTP Post failed: %v", err)
		return &HttpResult{Err: err, Success: false}
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Non-OK response: %d", resp.StatusCode)
		return &HttpResult{Success: false, StatusCode: resp.StatusCode}
	}

	log.Printf("[INFO] Async HTTP response received")

	return &HttpResult{
		StatusCode: resp.StatusCode,
		Success:    true,
	}
}

func (callAsync *CallAsyncLoopAppend) BuildFunctionUrl(gatewayAddr string, fnName string) string {
	return fmt.Sprintf("http://%s/asyncFunction/%s", gatewayAddr, fnName)
}
