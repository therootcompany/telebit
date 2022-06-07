package authutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type SuccessResponse struct {
	Success bool `json:"success"`
}

// Request makes an HTTP request the way we like...
func Request(method, fullurl, token string, payload io.Reader) (io.Reader, error) {
	HTTPClient := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest(method, fullurl, payload)
	if err != nil {
		return nil, err
	}
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if nil != payload {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%d: failed to read response body: %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%d: request failed: %v", resp.StatusCode, string(body))
	}

	return bytes.NewBuffer(body), nil
}

func Ping(authURL, token string) error {
	msg, err := Request("POST", authURL+"/ping", token, nil)
	if nil != err {
		return err
	}
	if nil == msg {
		return fmt.Errorf("invalid response")
	}
	resp := SuccessResponse{}
	err = json.NewDecoder(msg).Decode(&resp)
	if err != nil {
		return err
	}
	if true != resp.Success {
		return fmt.Errorf("expected successful response")
	}
	return nil
}
