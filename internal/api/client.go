package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const baseURL = "https://api.bitbucket.org/2.0"

type Client struct {
	email      string
	token      string
	httpClient *http.Client
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

type Page[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next"`
	Size   int    `json:"size"`
}

func NewClient(email, token string) *Client {
	return &Client{
		email: email,
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) newRequest(method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	fullURL := baseURL + path
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("User-Agent", "bb-cli/0.1.0")
	return req, nil
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 401 {
		return nil, &APIError{StatusCode: 401, Message: "authentication failed — run: bb auth login"}
	}
	if resp.StatusCode == 404 {
		return nil, &APIError{StatusCode: 404, Message: "not found"}
	}
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		msg := string(data)
		if json.Unmarshal(data, &errResp) == nil && errResp.Error.Message != "" {
			msg = errResp.Error.Message
		}
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, &APIError{StatusCode: resp.StatusCode, Message: msg}
	}

	return data, nil
}

func (c *Client) Get(path string, params url.Values) ([]byte, error) {
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

// GetRaw fetches a resource that returns non-JSON content (e.g., PR diffs).
func (c *Client) GetRaw(path string) (string, error) {
	req, err := c.newRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "*/*")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", &APIError{StatusCode: resp.StatusCode, Message: string(data[:min(len(data), 200)])}
	}
	return string(data), nil
}

func (c *Client) Post(path string, body any) ([]byte, error) {
	req, err := c.newRequest("POST", path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) Delete(path string) error {
	req, err := c.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	data, _ := io.ReadAll(resp.Body)
	return &APIError{StatusCode: resp.StatusCode, Message: string(data[:min(len(data), 200)])}
}

// Paginate follows Bitbucket's "next" pagination links, collecting all values up to max.
func Paginate[T any](c *Client, path string, params url.Values, max int) ([]T, error) {
	var all []T
	if params == nil {
		params = url.Values{}
	}
	if params.Get("pagelen") == "" {
		params.Set("pagelen", "50")
	}

	currentPath := path
	currentParams := params

	for {
		data, err := c.Get(currentPath, currentParams)
		if err != nil {
			return nil, err
		}

		var page Page[T]
		if err := json.Unmarshal(data, &page); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		all = append(all, page.Values...)

		if page.Next == "" || len(all) >= max {
			break
		}

		// Parse the next URL — strip base URL, keep path+query
		parsed, err := url.Parse(page.Next)
		if err != nil {
			break
		}
		currentPath = parsed.Path
		// Strip /2.0 prefix since our Get adds baseURL
		currentPath = strings.TrimPrefix(currentPath, "/2.0")
		currentParams = parsed.Query()
	}

	if len(all) > max {
		return all[:max], nil
	}
	return all, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
