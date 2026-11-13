package platformService

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"sync"
	"time"
)

const statusEndpoint = "https://status.bassaps.com/api/v1/status"
const releaseEndpoint = "https://download.bassaps.com/api/apps/launcher/latest"
const downloadURL = "https://download.bassaps.com/"
const maxBody = 1 << 20
const requestTimeout = 10 * time.Second

var errRedirect = errors.New("platform redirect rejected")

type requestError struct{ code string }

func (e *requestError) Error() string { return "platform: " + e.code }
func failure(code string) error       { return &requestError{code: code} }

type flight struct {
	done    chan struct{}
	cancel  context.CancelFunc
	waiters int
	body    []byte
	err     error
}

type PlatformService struct {
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	closed    bool
	client    *http.Client
	transport *http.Transport
	flights   map[string]*flight
	workers   sync.WaitGroup
	now       func() time.Time
}

func New() *PlatformService {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 8 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          4,
		MaxIdleConnsPerHost:   2,
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &PlatformService{
		ctx: ctx, cancel: cancel, transport: transport, now: time.Now,
		flights: make(map[string]*flight),
		client: &http.Client{Transport: transport, Timeout: requestTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirect }},
	}
}

// GetStatus reads the fixed public status endpoint. Wails supplies the call context.
func (s *PlatformService) GetStatus(ctx context.Context) (StatusSnapshot, error) {
	body, err := s.get(ctx, statusEndpoint)
	if err != nil {
		return StatusSnapshot{}, err
	}
	return parseStatus(body)
}

// GetLatestRelease returns public metadata for display; it never fetches an asset.
func (s *PlatformService) GetLatestRelease(ctx context.Context) (ReleaseSnapshot, error) {
	body, err := s.get(ctx, releaseEndpoint)
	if err != nil {
		return ReleaseSnapshot{}, err
	}
	return parseRelease(body, s.now())
}

func (s *PlatformService) get(ctx context.Context, endpoint string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, classifyError(ctx.Err())
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, failure("shutdown")
	}
	f := s.flights[endpoint]
	if f == nil {
		requestCtx, cancel := context.WithTimeout(s.ctx, requestTimeout)
		f = &flight{done: make(chan struct{}), cancel: cancel}
		s.flights[endpoint] = f
		s.workers.Add(1)
		go s.fetch(requestCtx, endpoint, f)
	}
	f.waiters++
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		f.waiters--
		if f.waiters == 0 && s.flights[endpoint] == f {
			delete(s.flights, endpoint)
			f.cancel()
		}
	}()
	select {
	case <-ctx.Done():
		return nil, classifyError(ctx.Err())
	case <-f.done:
		if ctx.Err() != nil {
			return nil, classifyError(ctx.Err())
		}
		return f.body, f.err
	}
}

func (s *PlatformService) fetch(ctx context.Context, endpoint string, f *flight) {
	defer s.workers.Done()
	defer f.cancel()
	body, err := s.read(ctx, endpoint)
	s.mu.Lock()
	f.body, f.err = body, err
	if s.flights[endpoint] == f {
		delete(s.flights, endpoint)
	}
	close(f.done)
	s.mu.Unlock()
}

func (s *PlatformService) read(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, failure("request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "BASSAPSLauncher-PublicInfo/1")
	response, err := s.client.Do(req)
	if err != nil {
		return nil, classifyError(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, failure(fmt.Sprintf("http_%d", response.StatusCode))
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return nil, failure("content_type")
	}
	if response.ContentLength > maxBody {
		return nil, failure("body_too_large")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return nil, classifyError(err)
	}
	if len(body) > maxBody {
		return nil, failure("body_too_large")
	}
	return body, nil
}

func classifyError(err error) error {
	var network net.Error
	var authority x509.UnknownAuthorityError
	var hostname x509.HostnameError
	var certificate x509.CertificateInvalidError
	var record tls.RecordHeaderError
	switch {
	case errors.Is(err, errRedirect):
		return failure("redirect")
	case errors.Is(err, context.Canceled):
		return failure("cancelled")
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &network) && network.Timeout():
		return failure("timeout")
	case errors.As(err, &authority), errors.As(err, &hostname), errors.As(err, &certificate), errors.As(err, &record):
		return failure("tls")
	default:
		return failure("connection")
	}
}

// ServiceShutdown only cancels requests and connections owned by this service.
func (s *PlatformService) ServiceShutdown() error {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.workers.Wait()
	s.transport.CloseIdleConnections()
	return nil
}
