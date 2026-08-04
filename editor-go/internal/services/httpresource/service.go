package httpresource

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type HTTPResourceService struct {
	allowedOrigins map[string]bool
	mu             sync.RWMutex
	client         *http.Client
}

func NewHTTPResourceService() *HTTPResourceService {
	return &HTTPResourceService{
		allowedOrigins: make(map[string]bool),
		client:         &http.Client{},
	}
}

func (s *HTTPResourceService) AllowOrigin(origin string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.allowedOrigins[origin] = true
}

func (s *HTTPResourceService) IsOriginAllowed(origin string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.allowedOrigins[origin]
}

func (s *HTTPResourceService) Fetch(url string, headers map[string]string) ([]byte, string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return body, resp.Header.Get("Content-Type"), nil
}

func (s *HTTPResourceService) FetchText(url string) (string, error) {
	body, _, err := s.Fetch(url, nil)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *HTTPResourceService) FetchJSON(url string, target interface{}) error {
	body, _, err := s.Fetch(url, nil)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func (s *HTTPResourceService) ProxyRequest(targetURL string, headers http.Header) ([]byte, http.Header, int, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, nil, 0, err
	}
	req.Header = headers.Clone()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, 0, err
	}

	respHeaders := make(http.Header)
	for key, vals := range resp.Header {
		if !strings.EqualFold(key, "transfer-encoding") &&
			!strings.EqualFold(key, "connection") &&
			!strings.EqualFold(key, "keep-alive") {
			respHeaders[key] = vals
		}
	}

	return body, respHeaders, resp.StatusCode, nil
}

func (s *HTTPResourceService) ValidateURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	if len(url) > 2048 {
		return fmt.Errorf("URL too long")
	}
	return nil
}