---
title: "Introduction"
description: "What jsonplaceholder is and how it is put together."
weight: 10
---

CLI for JSONPlaceholder fake REST API

jsonplaceholder is a single binary. It speaks to jsonplaceholder over plain HTTPS,
shapes the responses into clean records, and gets out of your way. There is
nothing to sign up for and nothing to run alongside it.

## How it is built

- A **library package** (`jsonplaceholder`) holds the HTTP client and the typed
  data models. It paces requests, sets an honest User-Agent, and retries the
  transient failures any public site throws under load.
- A **domain** (`jsonplaceholder/domain.go`) declares each operation once on the
  [any-cli/kit](https://github.com/tamnd/any-cli) framework. That single
  declaration becomes a CLI command, an HTTP route, an MCP tool, and a
  resource-URI dereference. It is the one place you add to the tool.
- A thin **`cmd/jsonplaceholder`** hands the assembled app to `kit.Run`, which
  builds the command tree and the serve and mcp surfaces.

## One operation, four surfaces

Because an operation is surface-neutral, the same `page` you run on the command
line is also a route and a tool:

```bash
jsonplaceholder page <path>                  # the command
jsonplaceholder serve --addr :7777           # GET /v1/page/<path>
jsonplaceholder mcp                          # the page tool, over stdio
ant get jsonplaceholder://page/<path>        # the URI dereference (via a host)
```

You write the fetch and the record shape; the surfaces come for free.

## Scope

jsonplaceholder is a read-only client over data jsonplaceholder already serves
publicly. It reads that data and shapes it for you. That narrow scope keeps it a
single small binary with no database, no daemon, and no setup.

Next: [install it](/getting-started/installation/), then take the
[quick start](/getting-started/quick-start/).
