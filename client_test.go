package fluent_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/devem-tech/fluent"
)

const baseURL = "https://jsonplaceholder.typicode.com"

type Post struct {
	UserID int    `json:"userId"`
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func newClient() *fluent.Client {
	return fluent.New().
		BaseURL(baseURL).
		HTTPClient(&http.Client{Timeout: 10 * time.Second})
}

func TestJSONPlaceholder_GetPostByID_Into(t *testing.T) {
	t.Parallel()

	resp := newClient().Get(context.Background(), "/posts/1")

	post, err := fluent.Into[Post](resp)
	if err != nil {
		t.Fatalf("Into returned error: %v", err)
	}

	if post.ID != 1 {
		t.Fatalf("expected ID=1, got %d", post.ID)
	}

	if post.Title == "" {
		t.Fatal("expected non-empty Title")
	}
}

func TestJSONPlaceholder_GetPostsWithQueryParam(t *testing.T) {
	t.Parallel()

	resp := newClient().
		Query("userId", "1").
		Get(context.Background(), "/posts")

	posts, err := fluent.Into[[]Post](resp)
	if err != nil {
		t.Fatalf("Into returned error: %v", err)
	}

	if len(posts) == 0 {
		t.Fatal("expected non-empty posts list")
	}

	for _, p := range posts {
		if p.UserID != 1 {
			t.Fatalf("expected all posts to have UserID=1, got %d (post ID=%d)", p.UserID, p.ID)
		}
	}
}

func TestJSONPlaceholder_InvalidPath_ReturnsErrNotOK(t *testing.T) {
	t.Parallel()

	resp := newClient().Get(context.Background(), "/this-path-should-not-exist")

	err := resp.Error()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, fluent.ErrNotOK) {
		t.Fatalf("expected ErrNotOK, got: %v", err)
	}
}

func TestJSONPlaceholder_Post_Created_Into(t *testing.T) {
	t.Parallel()

	resp := newClient().
		Body(map[string]any{
			"title":  "foo",
			"body":   "bar",
			"userId": 1,
		}).
		Post(context.Background(), "/posts")

	created, err := fluent.Into[Post](resp)
	if err != nil {
		t.Fatalf("Into returned error: %v", err)
	}

	if created.Title != "foo" {
		t.Fatalf("expected Title=foo, got %q", created.Title)
	}

	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
}

func TestJSONPlaceholder_Error_IncludesBodySnippet(t *testing.T) {
	t.Parallel()

	resp := newClient().Get(context.Background(), "/posts/0") // 404

	err := resp.Error()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, fluent.ErrNotOK) {
		t.Fatalf("expected ErrNotOK, got: %v", err)
	}

	var he *fluent.HTTPError
	if !errors.As(err, &he) {
		t.Fatalf("expected *HTTPError, got: %T (%v)", err, err)
	}

	if he.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code 404, got %d", he.StatusCode)
	}

	// body может быть {} или пустым, но сам факт, что поле доступно — важен
	if he.Body == nil {
		t.Fatal("expected Body to be non-nil")
	}
}
