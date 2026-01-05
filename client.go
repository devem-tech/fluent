package fluent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ErrNotOK возвращается, если сервер ответил не 2xx.
var ErrNotOK = errors.New("invalid status code")

type HTTPError struct {
	StatusCode int
	Status     string
	Method     string
	URL        string
	Body       []byte
}

func (e *HTTPError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("%s %s: %s", e.Method, e.URL, e.Status)
	}

	return fmt.Sprintf("%s %s: %s: %s", e.Method, e.URL, e.Status, string(e.Body))
}

func (e *HTTPError) Unwrap() error {
	return ErrNotOK
}

// httpClient — интерфейс для любого http-клиента, поддерживающего метод Do.
// Обычно это *http.Client.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client реализует chainable HTTP-клиент с поддержкой кастомного клиента, query-параметров, заголовков и JSON body.
type Client struct {
	baseURL string
	params  url.Values
	headers http.Header
	client  httpClient
	body    any
}

// New создает новый fluent-клиент с пустым baseURL и стандартными параметрами.
func New() *Client {
	return &Client{
		params:  make(url.Values),
		headers: make(http.Header),
		client:  http.DefaultClient,
	}
}

// BaseURL задает базовый адрес для всех последующих запросов.
// Если baseURL не задан, путь передается как абсолютный URL в метод Get или Post.
func (c *Client) BaseURL(baseURL string) *Client {
	c.baseURL = baseURL

	return c
}

// Query добавляет query-параметр к следующему запросу.
// Можно вызывать несколько раз для добавления разных параметров.
func (c *Client) Query(key, value string) *Client {
	c.params.Add(key, value)

	return c
}

// Header добавляет HTTP-заголовок к следующему запросу.
// Можно вызывать несколько раз для добавления разных заголовков.
func (c *Client) Header(key, value string) *Client {
	c.headers.Add(key, value)

	return c
}

// HTTPClient задает кастомный http-клиент (например, с таймаутом или прокси).
func (c *Client) HTTPClient(client httpClient) *Client {
	c.client = client

	return c
}

// Body задает тело запроса, которое будет сериализовано в JSON при отправке POST/PUT/PATCH/DELETE.
// Можно передавать любую структуру с json-тегами.
func (c *Client) Body(body any) *Client {
	c.body = body

	return c
}

// Reset очищает все query-параметры, заголовки и тело клиента.
func (c *Client) Reset() *Client {
	c.params = make(url.Values)
	c.headers = make(http.Header)
	c.body = nil

	return c
}

// Get выполняет HTTP GET-запрос по указанному пути или URL.
// Все добавленные query-параметры и заголовки будут включены в запрос.
// Если baseURL не задан, path должен быть абсолютным URL.
// Возвращает Response, оборачивающий http.Response и ошибку.
func (c *Client) Get(ctx context.Context, path string) *Response {
	return c.do(ctx, http.MethodGet, path)
}

// Post выполняет HTTP POST-запрос по указанному пути или URL.
// Все добавленные query-параметры и заголовки будут включены в запрос.
// Если передан body (метод Body), он будет сериализован в JSON.
// Если baseURL не задан, path должен быть абсолютным URL.
// Возвращает Response, оборачивающий http.Response и ошибку.
func (c *Client) Post(ctx context.Context, path string) *Response {
	return c.do(ctx, http.MethodPost, path)
}

// do выполняет HTTP-запрос с любым методом (GET, POST и др.).
func (c *Client) do(ctx context.Context, method, path string) *Response { //nolint:cyclop
	fullURL, err := c.fullURL(path)
	if err != nil {
		return &Response{err: err}
	}

	var body io.Reader
	if c.body != nil {
		b, err := json.Marshal(c.body)
		if err != nil {
			return &Response{err: err}
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return &Response{err: err}
	}

	// Если есть body, Content-Type JSON по умолчанию (если не переопределили)
	if c.body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range c.headers {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return &Response{err: err}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return &Response{err: err}
		}

		return &Response{
			err: &HTTPError{
				StatusCode: resp.StatusCode,
				Status:     resp.Status,
				Method:     method,
				URL:        fullURL,
				Body:       body,
			},
		}
	}

	// Сбросить body, чтобы оно не попало случайно в следующий запрос
	c.body = nil

	return &Response{resp: resp}
}

// fullURL формирует финальный URL с учетом baseURL, path и query-параметров.
// Если baseURL пустой, path должен быть абсолютным URL.
// Query-параметры из path будут дополнены параметрами из клиента (Query).
func (c *Client) fullURL(path string) (string, error) {
	if c.baseURL == "" {
		u, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}

		q := u.Query()

		for k, vals := range c.params {
			for _, v := range vals {
				q.Add(k, v)
			}
		}

		u.RawQuery = q.Encode()

		return u.String(), nil
	}

	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid baseURL: %w", err)
	}

	u.Path = strings.TrimSuffix(u.Path, "/") + "/" + strings.TrimPrefix(path, "/")
	u.RawQuery = c.params.Encode()

	return u.String(), nil
}
