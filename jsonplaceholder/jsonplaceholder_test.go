package jsonplaceholder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/jsonplaceholder-cli/jsonplaceholder"
)

func newTestClient(ts *httptest.Server) *jsonplaceholder.Client {
	cfg := jsonplaceholder.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return jsonplaceholder.NewClient(cfg)
}

func TestUserAgent(t *testing.T) {
	var gotUA string
	posts := []map[string]any{
		{"id": 1, "userId": 1, "title": "test", "body": "body"},
	}
	b, _ := json.Marshal(posts)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Posts(context.Background(), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("User-Agent header not set")
	}
	if !strings.Contains(gotUA, "jsonplaceholder-cli") {
		t.Errorf("User-Agent %q does not contain 'jsonplaceholder-cli'", gotUA)
	}
}

func TestParsePosts(t *testing.T) {
	posts := []map[string]any{
		{"id": 1, "userId": 1, "title": "Title One", "body": "Body one"},
		{"id": 2, "userId": 1, "title": "Title Two", "body": "Body two"},
	}
	b, _ := json.Marshal(posts)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	result, err := c.Posts(context.Background(), 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d posts, want 2", len(result))
	}
	if result[0].ID != 1 {
		t.Errorf("post[0].ID = %d, want 1", result[0].ID)
	}
	if result[0].UserID != 1 {
		t.Errorf("post[0].UserID = %d, want 1", result[0].UserID)
	}
	if result[0].Title != "Title One" {
		t.Errorf("post[0].Title = %q, want Title One", result[0].Title)
	}
}

func TestParseUser(t *testing.T) {
	user := map[string]any{
		"id":       1,
		"name":     "Leanne Graham",
		"username": "Bret",
		"email":    "Sincere@april.biz",
		"phone":    "1-770-736-8031 x56442",
		"website":  "hildegard.org",
		"address": map[string]any{
			"street":  "Kulas Light",
			"suite":   "Apt. 556",
			"city":    "Gwenborough",
			"zipcode": "92998-3874",
		},
		"company": map[string]any{
			"name": "Romaguera-Crona",
		},
	}
	b, _ := json.Marshal(user)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	result, err := c.User(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "Leanne Graham" {
		t.Errorf("Name = %q, want Leanne Graham", result.Name)
	}
	if result.Email != "Sincere@april.biz" {
		t.Errorf("Email = %q, want Sincere@april.biz", result.Email)
	}
	if result.City != "Gwenborough" {
		t.Errorf("City = %q, want Gwenborough", result.City)
	}
	if result.Company != "Romaguera-Crona" {
		t.Errorf("Company = %q, want Romaguera-Crona", result.Company)
	}
}

func TestParseTodos(t *testing.T) {
	todos := []map[string]any{
		{"id": 1, "userId": 1, "title": "delectus aut autem", "completed": false},
		{"id": 2, "userId": 1, "title": "quis ut nam facilis et officia qui", "completed": false},
	}
	b, _ := json.Marshal(todos)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	result, err := c.Todos(context.Background(), 1, 2, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d todos, want 2", len(result))
	}
	if result[0].Completed != false {
		t.Errorf("todo[0].Completed = %v, want false", result[0].Completed)
	}
	if result[0].Title != "delectus aut autem" {
		t.Errorf("todo[0].Title = %q, want delectus aut autem", result[0].Title)
	}
}

func TestLimitParam(t *testing.T) {
	var gotQuery string
	posts := []map[string]any{{"id": 1, "userId": 1, "title": "t", "body": "b"}}
	b, _ := json.Marshal(posts)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	c := newTestClient(ts)
	_, err := c.Posts(context.Background(), 0, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "_limit=5") {
		t.Errorf("query %q does not contain _limit=5", gotQuery)
	}
}

func TestRetry503(t *testing.T) {
	var hits int
	posts := []map[string]any{{"id": 1, "userId": 1, "title": "t", "body": "b"}}
	b, _ := json.Marshal(posts)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer ts.Close()

	cfg := jsonplaceholder.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := jsonplaceholder.NewClient(cfg)
	_, err := c.Posts(context.Background(), 0, 0)
	if err != nil {
		t.Fatalf("unexpected error after retry: %v", err)
	}
	if hits < 3 {
		t.Errorf("expected at least 3 hits (retries), got %d", hits)
	}
}
