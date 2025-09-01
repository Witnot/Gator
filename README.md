# Gator CLI

Gator is a simple CLI tool for managing RSS feeds, fetching posts, and browsing them.

---

## Prerequisites

Before using Gator, make sure you have the following installed:
- [PostgreSQL](https://www.postgresql.org/download/) (v12+ recommended)
- [Go](https://golang.org/dl/) (v1.21+ recommended)

---

## Installation

First, clone the repository:

```bash
git clone https://github.com/Witnot/Gator.git
cd Gator
```

You can install the `gator` CLI using `go install`:

```bash
# From the root of the project
go install ./...
```

This will compile the project and place the binary in your $GOPATH/bin (usually $HOME/go/bin).

Make sure that directory is in your PATH:

```bash
export PATH=$PATH:$HOME/go/bin
```

After that, you can run gator from anywhere:

```bash
gator reset
gator register lane
gator login lane
```

## Configuration

Gator uses a configuration file stored at:

```bash
$HOME/.gatorconfig.json
```

You can create it manually or let the CLI create it via commands like register and login. The config file looks like this:

```json
{
  "db_url": "postgres://postgres:postgres@localhost:5432/gator?sslmode=disable",
  "current_user_name": "boots"
}
```

- `db_url`: The Postgres connection string.
- `current_user_name`: The currently logged-in user.

## Usage

Here are a few commands you can run:

- `gator reset` — Clears all data and resets the database.
- `gator register <username>` — Creates a new user.
- `gator login <username>` — Logs in as an existing user.
- `gator users` — Lists all registered users (shows current user).
- `gator addfeed <name> <url>` — Adds a new feed and follows it automatically.
- `gator feeds` — Lists all feeds in the system.
- `gator follow <feed_url>` — Follow an existing feed.
- `gator following` — List feeds the current user is following.
- `gator browse [limit]` — Show recent posts for the current user (default limit: 2).
- `gator agg <duration>` — Starts fetching feeds in a loop every `<duration>` (e.g., 1m, 10s).

## Example Workflow

```bash
# Clone the repository
git clone https://github.com/Witnot/Gator.git
cd Gator

# Install the CLI
go install ./...

# Reset database
gator reset

# Create a new user
gator register lane

# Login as the user
gator login lane

# Add a feed
gator addfeed "TechCrunch" "https://techcrunch.com/feed/"

# Fetch new posts immediately
gator agg 10s

# Browse recent posts
gator browse 5
```
