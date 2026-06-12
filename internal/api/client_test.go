package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// newTestClient creates a Client that points to the given test server.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c := NewClient("test@example.com", "tok123")
	c.httpClient = srv.Client()
	// Override baseURL by patching requests through a custom transport
	// that rewrites the URL to the test server.
	c.httpClient.Transport = &rewriteTransport{base: srv.URL, wrapped: srv.Client().Transport}
	return c
}

type rewriteTransport struct {
	base    string
	wrapped http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace baseURL prefix with test server URL
	req.URL.Scheme = "http"
	parsed, _ := url.Parse(rt.base)
	req.URL.Host = parsed.Host
	// Strip /2.0 prefix that newRequest adds via baseURL concat
	req.URL.Path = strings.TrimPrefix(req.URL.Path, "/2.0")
	if rt.wrapped != nil {
		return rt.wrapped.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

func TestAPIError_Error(t *testing.T) {
	e := &APIError{StatusCode: 403, Message: "forbidden"}
	if got := e.Error(); got != "API error 403: forbidden" {
		t.Errorf("APIError.Error() = %q", got)
	}
}

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if !strings.Contains(r.Header.Get("Authorization"), "Basic") {
			t.Error("missing Basic auth")
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	data, err := c.Get("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body = %q", string(data))
	}
}

func TestGet_WithParams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("foo") != "bar" {
			t.Errorf("missing query param foo=bar, got %s", r.URL.RawQuery)
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Get("/test", url.Values{"foo": {"bar"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestGet_401Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Get("/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
}

func TestGet_404Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Get("/test", nil)
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.StatusCode != 404 {
		t.Errorf("expected 404 APIError, got %v", err)
	}
}

func TestGet_GenericError_ParsesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "bad field"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Get("/test", nil)
	apiErr := err.(*APIError)
	if apiErr.Message != "bad field" {
		t.Errorf("message = %q, want \"bad field\"", apiErr.Message)
	}
}

func TestGet_TruncatesLongError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(strings.Repeat("x", 300)))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	_, err := c.Get("/test", nil)
	apiErr := err.(*APIError)
	if len(apiErr.Message) > 200 {
		t.Errorf("message length = %d, want <= 200", len(apiErr.Message))
	}
}

func TestGetRaw_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("diff --git a/file"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	got, err := c.GetRaw("/diff")
	if err != nil {
		t.Fatal(err)
	}
	if got != "diff --git a/file" {
		t.Errorf("GetRaw() = %q", got)
	}
}

func TestDelete_204Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.Delete("/test"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}

func TestDelete_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.Delete("/test")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 403 {
		t.Errorf("status = %d, want 403", apiErr.StatusCode)
	}
}

func TestPost_SendsJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing Content-Type: application/json")
		}
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Write([]byte(`{"id":1}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	data, err := c.Post("/test", map[string]string{"title": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"id":1`) {
		t.Errorf("Post() = %q", string(data))
	}
}

func TestPaginate_SinglePage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Page[map[string]int]{
			Values: []map[string]int{{"n": 1}, {"n": 2}},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	results, err := Paginate[map[string]int](c, "/items", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestPaginate_MultiplePages(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			json.NewEncoder(w).Encode(Page[map[string]int]{
				Values: []map[string]int{{"n": 1}},
				Next:   fmt.Sprintf("http://%s/items?page=2", r.Host),
			})
		} else {
			json.NewEncoder(w).Encode(Page[map[string]int]{
				Values: []map[string]int{{"n": 2}},
			})
		}
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	results, err := Paginate[map[string]int](c, "/items", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestPaginate_RespectsMax(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(Page[map[string]int]{
			Values: []map[string]int{{"n": 1}, {"n": 2}, {"n": 3}},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	results, err := Paginate[map[string]int](c, "/items", nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}
