// Package jsonplaceholder is the library behind the jsonplaceholder command line:
// the HTTP client, request shaping, and the typed data models for the
// JSONPlaceholder fake REST API at https://jsonplaceholder.typicode.com.
//
// The API is public and requires no key. It provides fake posts, comments,
// users, todos, albums, and photos for testing and prototyping.
package jsonplaceholder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Host is the JSONPlaceholder API hostname.
const Host = "jsonplaceholder.typicode.com"

// DefaultUserAgent identifies the client to the API.
const DefaultUserAgent = "jsonplaceholder-cli/0.1.0 (github.com/tamnd/jsonplaceholder-cli)"

// Config holds constructor parameters for Client.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Timeout   time.Duration
	Retries   int
}

// DefaultConfig returns production defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://" + Host,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Timeout:   30 * time.Second,
		Retries:   3,
	}
}

// Client is the JSONPlaceholder HTTP client.
type Client struct {
	cfg  Config
	http *http.Client
	mu   sync.Mutex
	last time.Time
}

// NewClient constructs a Client from cfg.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Post is a JSONPlaceholder blog post.
type Post struct {
	ID     int    `json:"id"      kit:"id"`
	UserID int    `json:"user_id"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// Comment is a comment on a post.
type Comment struct {
	ID     int    `json:"id"      kit:"id"`
	PostID int    `json:"post_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Body   string `json:"body"`
}

// User is a JSONPlaceholder user with flattened address and company fields.
type User struct {
	ID       int    `json:"id"       kit:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Website  string `json:"website"`
	City     string `json:"city"`    // from address.city
	Company  string `json:"company"` // from company.name
}

// Todo is a task item.
type Todo struct {
	ID        int    `json:"id"         kit:"id"`
	UserID    int    `json:"user_id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// Album is a photo album.
type Album struct {
	ID     int    `json:"id"      kit:"id"`
	UserID int    `json:"user_id"`
	Title  string `json:"title"`
}

// Photo is a photo within an album.
type Photo struct {
	ID           int    `json:"id"            kit:"id"`
	AlbumID      int    `json:"album_id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url"`
}

// wirePost is the JSON shape from the API (camelCase keys).
type wirePost struct {
	ID     int    `json:"id"`
	UserID int    `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

func (w wirePost) toPost() Post {
	return Post{ID: w.ID, UserID: w.UserID, Title: w.Title, Body: w.Body}
}

// wireTodo is the JSON shape from the API (camelCase keys).
type wireTodo struct {
	ID        int    `json:"id"`
	UserID    int    `json:"userId"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func (w wireTodo) toTodo() Todo {
	return Todo{ID: w.ID, UserID: w.UserID, Title: w.Title, Completed: w.Completed}
}

// wireComment is the JSON shape from the API (camelCase keys).
type wireComment struct {
	ID     int    `json:"id"`
	PostID int    `json:"postId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Body   string `json:"body"`
}

func (w wireComment) toComment() Comment {
	return Comment{ID: w.ID, PostID: w.PostID, Name: w.Name, Email: w.Email, Body: w.Body}
}

// wireAlbum is the JSON shape from the API (camelCase keys).
type wireAlbum struct {
	ID     int    `json:"id"`
	UserID int    `json:"userId"`
	Title  string `json:"title"`
}

func (w wireAlbum) toAlbum() Album {
	return Album{ID: w.ID, UserID: w.UserID, Title: w.Title}
}

// wirePhoto is the JSON shape from the API (camelCase keys).
type wirePhoto struct {
	ID           int    `json:"id"`
	AlbumID      int    `json:"albumId"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

func (w wirePhoto) toPhoto() Photo {
	return Photo{ID: w.ID, AlbumID: w.AlbumID, Title: w.Title, URL: w.URL, ThumbnailURL: w.ThumbnailURL}
}

// wireUser is the JSON shape from the API, with nested address and company.
type wireUser struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Website  string `json:"website"`
	Address  struct {
		Street  string `json:"street"`
		Suite   string `json:"suite"`
		City    string `json:"city"`
		Zipcode string `json:"zipcode"`
	} `json:"address"`
	Company struct {
		Name string `json:"name"`
	} `json:"company"`
}

func (w wireUser) toUser() User {
	return User{
		ID:       w.ID,
		Name:     w.Name,
		Username: w.Username,
		Email:    w.Email,
		Phone:    w.Phone,
		Website:  w.Website,
		City:     w.Address.City,
		Company:  w.Company.Name,
	}
}

// Posts returns a list of posts filtered by optional userId and limit.
// Zero values are omitted from the query.
func (c *Client) Posts(ctx context.Context, userID, limit int) ([]Post, error) {
	params := map[string]string{}
	if userID > 0 {
		params["userId"] = strconv.Itoa(userID)
	}
	if limit > 0 {
		params["_limit"] = strconv.Itoa(limit)
	}
	body, err := c.get(ctx, c.buildURL("/posts", params))
	if err != nil {
		return nil, err
	}
	var wire []wirePost
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse posts: %w", err)
	}
	out := make([]Post, len(wire))
	for i, w := range wire {
		out[i] = w.toPost()
	}
	return out, nil
}

// GetPost returns a single post by id.
func (c *Client) GetPost(ctx context.Context, id int) (Post, error) {
	body, err := c.get(ctx, c.buildURL(fmt.Sprintf("/posts/%d", id), nil))
	if err != nil {
		return Post{}, err
	}
	var wire wirePost
	if err := json.Unmarshal(body, &wire); err != nil {
		return Post{}, fmt.Errorf("parse post: %w", err)
	}
	return wire.toPost(), nil
}

// Comments returns comments filtered by optional postId and limit.
func (c *Client) Comments(ctx context.Context, postID, limit int) ([]Comment, error) {
	params := map[string]string{}
	if postID > 0 {
		params["postId"] = strconv.Itoa(postID)
	}
	if limit > 0 {
		params["_limit"] = strconv.Itoa(limit)
	}
	body, err := c.get(ctx, c.buildURL("/comments", params))
	if err != nil {
		return nil, err
	}
	var wire []wireComment
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse comments: %w", err)
	}
	out := make([]Comment, len(wire))
	for i, w := range wire {
		out[i] = w.toComment()
	}
	return out, nil
}

// Users returns all users.
func (c *Client) Users(ctx context.Context) ([]User, error) {
	body, err := c.get(ctx, c.buildURL("/users", nil))
	if err != nil {
		return nil, err
	}
	var wire []wireUser
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse users: %w", err)
	}
	out := make([]User, len(wire))
	for i, w := range wire {
		out[i] = w.toUser()
	}
	return out, nil
}

// User returns a single user by id.
func (c *Client) User(ctx context.Context, id int) (User, error) {
	body, err := c.get(ctx, c.buildURL(fmt.Sprintf("/users/%d", id), nil))
	if err != nil {
		return User{}, err
	}
	var wire wireUser
	if err := json.Unmarshal(body, &wire); err != nil {
		return User{}, fmt.Errorf("parse user: %w", err)
	}
	return wire.toUser(), nil
}

// Todos returns todos filtered by optional userId, completed flag, and limit.
// completedSet indicates whether the completed filter should be applied.
func (c *Client) Todos(ctx context.Context, userID, limit int, completed, completedSet bool) ([]Todo, error) {
	params := map[string]string{}
	if userID > 0 {
		params["userId"] = strconv.Itoa(userID)
	}
	if completedSet {
		if completed {
			params["completed"] = "true"
		} else {
			params["completed"] = "false"
		}
	}
	if limit > 0 {
		params["_limit"] = strconv.Itoa(limit)
	}
	body, err := c.get(ctx, c.buildURL("/todos", params))
	if err != nil {
		return nil, err
	}
	var wire []wireTodo
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse todos: %w", err)
	}
	out := make([]Todo, len(wire))
	for i, w := range wire {
		out[i] = w.toTodo()
	}
	return out, nil
}

// Albums returns albums filtered by optional userId and limit.
func (c *Client) Albums(ctx context.Context, userID, limit int) ([]Album, error) {
	params := map[string]string{}
	if userID > 0 {
		params["userId"] = strconv.Itoa(userID)
	}
	if limit > 0 {
		params["_limit"] = strconv.Itoa(limit)
	}
	body, err := c.get(ctx, c.buildURL("/albums", params))
	if err != nil {
		return nil, err
	}
	var wire []wireAlbum
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse albums: %w", err)
	}
	out := make([]Album, len(wire))
	for i, w := range wire {
		out[i] = w.toAlbum()
	}
	return out, nil
}

// Photos returns photos filtered by optional albumId and limit.
func (c *Client) Photos(ctx context.Context, albumID, limit int) ([]Photo, error) {
	params := map[string]string{}
	if albumID > 0 {
		params["albumId"] = strconv.Itoa(albumID)
	}
	if limit > 0 {
		params["_limit"] = strconv.Itoa(limit)
	}
	body, err := c.get(ctx, c.buildURL("/photos", params))
	if err != nil {
		return nil, err
	}
	var wire []wirePhoto
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("parse photos: %w", err)
	}
	out := make([]Photo, len(wire))
	for i, w := range wire {
		out[i] = w.toPhoto()
	}
	return out, nil
}

// buildURL constructs a full API URL with query parameters.
func (c *Client) buildURL(path string, params map[string]string) string {
	base := strings.TrimRight(c.cfg.BaseURL, "/")
	if len(params) == 0 {
		return base + path
	}
	q := url.Values{}
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	return base + path + "?" + q.Encode()
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
