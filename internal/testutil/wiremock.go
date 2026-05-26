package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// WireMockContainer wraps a running WireMock container for testing.
type WireMockContainer struct {
	Container testcontainers.Container
	BaseURL   string
	AdminURL  string
}

// StartWireMock starts a WireMock container and returns the wrapper.
func StartWireMock(ctx context.Context, t *testing.T) *WireMockContainer {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "wiremock/wiremock:3.6.0",
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForHTTP("/__admin/mappings").WithPort("8080/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start WireMock container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	baseURL := fmt.Sprintf("http://%s:%s", host, port.Port())

	return &WireMockContainer{
		Container: container,
		BaseURL:   baseURL,
		AdminURL:  baseURL + "/__admin",
	}
}

// Stop terminates the WireMock container.
func (w *WireMockContainer) Stop(ctx context.Context) {
	if w.Container != nil {
		w.Container.Terminate(ctx)
	}
}

// Reset clears all stubs and request logs.
func (w *WireMockContainer) Reset(t *testing.T) {
	t.Helper()
	req, _ := http.NewRequest("POST", w.AdminURL+"/reset", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to reset WireMock: %v", err)
	}
	resp.Body.Close()
}

// StubMapping represents a WireMock stub configuration.
type StubMapping struct {
	ScenarioName    string       `json:"scenarioName,omitempty"`
	RequiredState   string       `json:"requiredScenarioState,omitempty"`
	NewState        string       `json:"newScenarioState,omitempty"`
	Request         StubRequest  `json:"request"`
	Response        StubResponse `json:"response"`
}

// StubRequest defines which requests to match.
type StubRequest struct {
	Method      string                       `json:"method"`
	URLPathPattern string                    `json:"urlPathPattern,omitempty"`
	URLPath     string                       `json:"urlPath,omitempty"`
	Headers     map[string]StubHeaderMatcher `json:"headers,omitempty"`
	BodyPatterns []StubBodyPattern           `json:"bodyPatterns,omitempty"`
}

// StubHeaderMatcher matches a header value.
type StubHeaderMatcher struct {
	EqualTo  string `json:"equalTo,omitempty"`
	Contains string `json:"contains,omitempty"`
}

// StubBodyPattern matches the request body.
type StubBodyPattern struct {
	EqualToJSON   interface{} `json:"equalToJson,omitempty"`
	MatchesJSONPath string    `json:"matchesJsonPath,omitempty"`
}

// StubResponse defines the response to return.
type StubResponse struct {
	Status      int               `json:"status"`
	Body        string            `json:"body,omitempty"`
	JSONBody    interface{}       `json:"jsonBody,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// RegisterStub adds a stub mapping to WireMock.
func (w *WireMockContainer) RegisterStub(t *testing.T, stub StubMapping) {
	t.Helper()

	body, err := json.Marshal(stub)
	if err != nil {
		t.Fatalf("failed to marshal stub: %v", err)
	}

	resp, err := http.Post(w.AdminURL+"/mappings", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to register stub: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("failed to register stub (status %d): %s", resp.StatusCode, string(respBody))
	}
}

// VerifyRequest represents the verification request to WireMock.
type VerifyRequest struct {
	Method     string `json:"method"`
	URLPath    string `json:"urlPath,omitempty"`
	URLPathPattern string `json:"urlPathPattern,omitempty"`
}

// VerifyResponse is WireMock's response to a verification request.
type VerifyResponse struct {
	Count    int `json:"count"`
	Requests []struct {
		Request struct {
			Method  string            `json:"method"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
			Body    string            `json:"body"`
		} `json:"request"`
	} `json:"requests"`
}

// VerifyCallCount asserts the number of times a specific endpoint was called.
func (w *WireMockContainer) VerifyCallCount(t *testing.T, method, urlPath string, expectedCount int) {
	t.Helper()

	verifyReq := VerifyRequest{
		Method:  method,
		URLPath: urlPath,
	}
	body, _ := json.Marshal(verifyReq)

	resp, err := http.Post(w.AdminURL+"/requests/count", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to verify calls: %v", err)
	}
	defer resp.Body.Close()

	var verifyResp struct {
		Count int `json:"count"`
	}
	json.NewDecoder(resp.Body).Decode(&verifyResp)

	if verifyResp.Count != expectedCount {
		t.Errorf("expected %d calls to %s %s, got %d", expectedCount, method, urlPath, verifyResp.Count)
	}
}

// GetRequests returns all recorded requests matching the given method and path.
func (w *WireMockContainer) GetRequests(t *testing.T, method, urlPath string) []RecordedRequest {
	t.Helper()

	findReq := VerifyRequest{
		Method:  method,
		URLPath: urlPath,
	}
	body, _ := json.Marshal(findReq)

	resp, err := http.Post(w.AdminURL+"/requests/find", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to get requests: %v", err)
	}
	defer resp.Body.Close()

	var findResp struct {
		Requests []RecordedRequest `json:"requests"`
	}
	json.NewDecoder(resp.Body).Decode(&findResp)

	return findResp.Requests
}

// RecordedRequest represents a request recorded by WireMock.
type RecordedRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// VerifyAuthHeader asserts that all requests to the given path included the expected auth header.
func (w *WireMockContainer) VerifyAuthHeader(t *testing.T, method, urlPath, expectedToken string) {
	t.Helper()

	requests := w.GetRequests(t, method, urlPath)
	for i, req := range requests {
		authHeader := req.Headers["Authorization"]
		expected := "Bearer " + expectedToken
		if authHeader != expected {
			t.Errorf("request %d to %s %s: expected Authorization %q, got %q", i, method, urlPath, expected, authHeader)
		}
	}
}
