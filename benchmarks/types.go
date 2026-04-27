package main

type InvokeRequest struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type InvokeResponse struct {
	ColdStart bool `json:"cold_start"`
}
