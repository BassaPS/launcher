// Package srtoolsrelay connects the SRTools website to one launcher-selected M7H
// endpoint. It is a loopback-only, origin-restricted relay, not a general proxy.
package srtoolsrelay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Origin = "https://srtools.punklorde.org"
const maxRequest = 4 << 20
const maxResponse = 8 << 20

type Relay struct {
	target  *url.URL
	client  *http.Client
	server  *http.Server
	baseURL string
	port    string
	slots   chan struct{}
}

func Start(base string) (*Relay, error) {
	target, err := url.Parse(strings.TrimSpace(base))
	if err != nil || target == nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "https") || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return nil, fmt.Errorf("invalid M7H server URL")
	}
	target.Path = strings.TrimRight(target.Path, "/")
	if !strings.HasSuffix(target.Path, "/sr-tools") {
		target.Path += "/sr-tools"
	}
	target.RawPath = ""
	listener, err := net.Listen("tcp", "127.0.0.1:24680")
	if err != nil {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return nil, err
	}
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil // Never pass account data through an ambient HTTP_PROXY.
	relay := &Relay{
		target: target, port: port, baseURL: "http://127.0.0.1:" + port,
		client: &http.Client{Transport: transport, Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		slots:  make(chan struct{}, 2),
	}
	relay.server = &http.Server{Handler: http.HandlerFunc(relay.serve), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() { _ = relay.server.Serve(listener) }()
	return relay, nil
}

func (r *Relay) URL() string { return r.baseURL }
func (r *Relay) Close() error {
	r.client.CloseIdleConnections()
	return r.server.Close()
}

func failure(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": status, "message": message})
}

func (r *Relay) serve(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	host, port, err := net.SplitHostPort(req.Host)
	if err != nil || port != r.port || (host != "127.0.0.1" && !strings.EqualFold(host, "localhost")) || req.Header.Get("Origin") != Origin {
		failure(w, http.StatusForbidden, "SRTools origin or local host is not allowed")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", Origin)
	w.Header().Set("Vary", "Origin")
	if req.URL.Path != "/sr-tools" || req.URL.RawQuery != "" {
		failure(w, 404, "Endpoint not allowed")
		return
	}
	if req.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if req.Header.Get("Access-Control-Request-Private-Network") == "true" {
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if req.Method != http.MethodPost {
		failure(w, 405, "Only POST is allowed")
		return
	}
	contentType, _, contentTypeErr := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if contentTypeErr != nil || contentType != "application/json" {
		failure(w, 415, "JSON is required")
		return
	}
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	default:
		failure(w, 429, "Relay is busy")
		return
	}
	requestBody, bodyErr := io.ReadAll(http.MaxBytesReader(w, req.Body, maxRequest))
	if bodyErr != nil {
		failure(w, 413, "Request is too large or unreadable")
		return
	}
	upstreamReq, err := http.NewRequestWithContext(req.Context(), http.MethodPost, r.target.String(), bytes.NewReader(requestBody))
	if err != nil {
		failure(w, 502, "Cannot build M7H request")
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Accept", "application/json")
	response, err := r.client.Do(upstreamReq)
	if err != nil {
		failure(w, 502, "Cannot reach the configured M7H server")
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponse+1))
	if err != nil || len(body) > maxResponse {
		failure(w, 502, "Invalid or oversized M7H response")
		return
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		failure(w, 502, "M7H redirect was not followed")
		return
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	// Preserve M7H's validation/authentication errors instead of hiding them as 502.
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(body)
}
