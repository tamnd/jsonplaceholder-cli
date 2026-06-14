package jsonplaceholder

import (
	"context"
	"net/url"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes jsonplaceholder as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/jsonplaceholder-cli/jsonplaceholder"
//
// The same Domain also builds the standalone jsonplaceholder binary (see cli.NewApp),
// so the binary and a host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the jsonplaceholder driver.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "jsonplaceholder",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "jsonplaceholder",
			Short:  "CLI for JSONPlaceholder fake REST API",
			Long: `CLI for JSONPlaceholder fake REST API

jsonplaceholder reads fake REST data over plain HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools. No API
key, nothing to run alongside it.`,
			Site: Host,
			Repo: "https://github.com/tamnd/jsonplaceholder-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "posts", Group: "read", List: true,
		Summary: "List posts", URIType: "post"}, listPosts)

	kit.Handle(app, kit.OpMeta{Name: "users", Group: "read", List: true,
		Summary: "List users", URIType: "user"}, listUsers)

	kit.Handle(app, kit.OpMeta{Name: "todos", Group: "read", List: true,
		Summary: "List todos", URIType: "todo"}, listTodos)

	kit.Handle(app, kit.OpMeta{Name: "comments", Group: "read", List: true,
		Summary: "List comments", URIType: "comment"}, listComments)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type postsRef struct {
	UserID int     `kit:"flag,inherit" name:"user-id" help:"filter by user ID"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type usersRef struct {
	Client *Client `kit:"inject"`
}

type todosRef struct {
	UserID int     `kit:"flag,inherit" name:"user-id" help:"filter by user ID"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type commentsRef struct {
	PostID int     `kit:"flag,inherit" name:"post-id" help:"filter by post ID"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func listPosts(ctx context.Context, in postsRef, emit func(*Post) error) error {
	posts, err := in.Client.Posts(ctx, in.UserID, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range posts {
		if err := emit(&posts[i]); err != nil {
			return err
		}
	}
	return nil
}

func listUsers(ctx context.Context, in usersRef, emit func(*User) error) error {
	users, err := in.Client.Users(ctx)
	if err != nil {
		return mapErr(err)
	}
	for i := range users {
		if err := emit(&users[i]); err != nil {
			return err
		}
	}
	return nil
}

func listTodos(ctx context.Context, in todosRef, emit func(*Todo) error) error {
	todos, err := in.Client.Todos(ctx, in.UserID, in.Limit, false, false)
	if err != nil {
		return mapErr(err)
	}
	for i := range todos {
		if err := emit(&todos[i]); err != nil {
			return err
		}
	}
	return nil
}

func listComments(ctx context.Context, in commentsRef, emit func(*Comment) error) error {
	comments, err := in.Client.Comments(ctx, in.PostID, in.Limit)
	if err != nil {
		return mapErr(err)
	}
	for i := range comments {
		if err := emit(&comments[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	id = canonPath(input)
	if id == "" {
		return "", "", errs.Usage("unrecognized jsonplaceholder reference: %q", input)
	}
	return "post", id, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "post", "user", "todo", "album", "photo", "comment":
		return "https://" + Host + "/" + uriType + "s/" + strings.Trim(id, "/"), nil
	default:
		return "", errs.Usage("jsonplaceholder has no resource type %q", uriType)
	}
}

// --- helpers ---

func canonPath(input string) string {
	input = strings.TrimSpace(input)
	if u, err := url.Parse(input); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		return strings.Trim(u.Path, "/")
	}
	return strings.Trim(input, "/")
}

func mapErr(err error) error {
	return err
}
