# Fluent HTTP Client for Go 🚀

`fluent` is a lightweight, zero-dependency Go library designed to make HTTP requests more readable and maintainable. It provides a chainable (fluent) API for building requests, handling JSON, and managing errors with ease.

## Features

- 🔗 Fluent & Chainable API
- 🚀 GET & POST Support
- 🛠 Flexible Configuration
- 📦 Automatic JSON Serialization
- 🧬 Generic Response Decoding
- 🔌 Custom `http.Client` support
- ⌛ Context-Aware Requests
- ⚠️ Detailed Error Handling
- 🪶 Zero Dependencies

## Installation

```bash
go get -u github.com/devem-tech/fluent
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/devem-tech/fluent"
)

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func main() {
	posts, err := fluent.Into[[]Post](
		fluent.New().
			BaseURL("https://jsonplaceholder.typicode.com").
			Query("userId", "1").
			Get(context.Background(), "/posts"),
	)
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range posts {
		fmt.Println(p.ID, p.Title)
	}
}
```

## Creating a Client

> [!IMPORTANT]
> The `Client` instance is **not thread-safe**. If you need to make concurrent requests from different goroutines,
> create a new client instance for each or use a mutex.

```go
c := fluent.New()
```

## Base URL

```go
c.BaseURL("https://jsonplaceholder.typicode.com")
```

If `BaseURL` is not set, you must pass a full URL into `Get` or `Post`.

## Query Parameters

```go
c.Query("userId", "1")
```

Query parameters are accumulated and applied to the next request(s) until you call `Reset()`.

## Headers

```go
c.Header("Accept", "application/json")
```

Headers are accumulated and applied to the next request(s) until you call `Reset()`.

## JSON Body (POST Example)

```go
resp := fluent.New().
	BaseURL("https://jsonplaceholder.typicode.com").
	Body(map[string]any{
		"title":  "foo",
		"body":   "bar",
		"userId": 1,
	}).
	Post(context.Background(), "/posts")
```

When `Body(...)` is set, the request body is serialized to JSON and `Content-Type: application/json` is set automatically.

## Decoding JSON Responses

`Into[T]` decodes the response body into a value of type `T` and closes the body automatically.

```go
type Post struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

post, err := fluent.Into[Post](resp)
```

## Accessing Raw Response Data

### Raw Bytes

```go
data, err := resp.Raw()
```

### Manual Body Reading

```go
body, err := resp.Body()
if err != nil {
	log.Fatal(err)
}
defer body.Close()

// Read body manually...
```

> [!NOTE]
> If you read the body manually, make sure you close it.  
> Otherwise, HTTP connections may not be reused efficiently.

## Error Handling

The client treats **any non-2xx response** as an error.

- `ErrNotOK` is a sentinel error you can match with `errors.Is`.
- `HTTPError` provides details: `StatusCode`, `Status`, `Method`, `URL`, and the response `Body`.

```go
if err := resp.Error(); err != nil {
	// handle error
}
```

### Inspecting HTTPError

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/devem-tech/fluent"
)

func main() {
	c := fluent.New().BaseURL("https://jsonplaceholder.typicode.com")

	// This endpoint does not exist => non-2xx error
	resp := c.Get(context.Background(), "/this-path-does-not-exist")

	err := resp.Error()
	if err == nil {
		log.Fatal("expected error")
	}

	if errors.Is(err, fluent.ErrNotOK) {
		fmt.Println("request failed with non-2xx status")
	}

	var he *fluent.HTTPError
	if errors.As(err, &he) {
		fmt.Println("status code:", he.StatusCode)
		fmt.Println("status:", he.Status)
		fmt.Println("method:", he.Method)
		fmt.Println("url:", he.URL)
		fmt.Println("body:", string(he.Body))
	}
}
```

## Resetting Client State

```go
c.Reset()
```

Clears query parameters, headers, and request body.

## Custom HTTP Client

```go
c.HTTPClient(&http.Client{
	Timeout: 5 * time.Second,
})
```

Useful for configuring timeouts, proxies, or transports.

## Notes for High Load Usage

- Prefer configuring timeouts on the underlying `http.Client`.
- Always ensure response bodies are closed (use `Into`/`Raw`, or `Body` + `Close`).

## License

MIT
