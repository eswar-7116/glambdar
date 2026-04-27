package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Client struct {
	BaseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

func (c *Client) Deploy(zipPath string, rateLimit int) error {
	file, err := os.Open(zipPath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(zipPath))
	io.Copy(part, file)
	writer.WriteField("rateLimit", strconv.Itoa(rateLimit))
	writer.Close()

	req, _ := http.NewRequest("POST", c.BaseURL+"/deploy", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *Client) Cleanup(funcName string) {
	req, _ := http.NewRequest("DELETE", c.BaseURL+"/del/"+funcName, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func (c *Client) MeasureInvoke(funcName string) (time.Duration, int) {
	start := time.Now()
	payload := InvokeRequest{Body: `{"test": true}`}
	jsonBody, _ := json.Marshal(payload)
	resp, err := http.Post(c.BaseURL+"/invoke/"+funcName, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return time.Since(start), resp.StatusCode
}

func (c *Client) MeasureInvokeWithColdStart(funcName string) (bool, int) {
	payload := InvokeRequest{Body: `{"test": true}`}
	jsonBody, _ := json.Marshal(payload)
	resp, err := http.Post(c.BaseURL+"/invoke/"+funcName, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()
	var res InvokeResponse
	json.NewDecoder(resp.Body).Decode(&res)
	return res.ColdStart, resp.StatusCode
}
