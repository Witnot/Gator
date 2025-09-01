package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"html"
    "strings"
	"github.com/Witnot/Gator/internal/config"
	"github.com/Witnot/Gator/internal/database"
	"github.com/google/uuid"
	"github.com/Witnot/Gator/internal/rss"	
	"strconv"
    "database/sql"	
)

// state holds the application state (config, db, etc.)
type state struct {
	cfg *config.Config
	db  *database.Queries
}

// command represents a CLI command with a name and args
type command struct {
	name string
	args []string
}

// handlerLogin handles the "login" command
func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: login requires a username")
		os.Exit(1)
	}

	username := cmd.args[0]

	// Check if user exists in the database
	user, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: user %q does not exist\n", username)
		os.Exit(1)
	}

	// Set current user in config
	if err := s.cfg.SetUser(user.Name); err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	fmt.Printf("Logged in as %q\n", user.Name)
	return nil
}


// handlerRegister handles the "register" command
func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("register requires a username")
	}

	username := cmd.args[0]

	// Build CreateUser params
	user := database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      username,
	}

	// Insert user
	createdUser, err := s.db.CreateUser(context.Background(), user)
	if err != nil {
		// Likely unique violation if user already exists
		fmt.Fprintln(os.Stderr, "Error: could not create user — maybe it already exists?")
		os.Exit(1)
	}

	// Set current user in config
	if err := s.cfg.SetUser(username); err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	// Success output
	fmt.Printf("User %q was created and set as the current user\n", createdUser.Name)
	log.Printf("DEBUG: created user: %+v\n", createdUser)

	return nil
}

func handlerReset(s *state, cmd command) error {
	// Execute the query
	if err := s.db.ResetUsers(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to reset database:", err)
		os.Exit(1)
	}

	fmt.Println("Database has been reset successfully.")
	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to get users:", err)
		os.Exit(1)
	}

	currentUser := s.cfg.CurrentUserName

	for _, u := range users {
		if u.Name == currentUser {
			fmt.Printf("* %s (current)\n", u.Name)
		} else {
			fmt.Printf("* %s\n", u.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
    if len(cmd.args) < 1 {
        return fmt.Errorf("usage: agg <time_between_reqs>")
    }

    // Step 1: Parse duration
    durationStr := cmd.args[0]
    timeBetweenRequests, err := time.ParseDuration(durationStr)
    if err != nil {
        return fmt.Errorf("invalid duration %q: %w", durationStr, err)
    }

    fmt.Printf("Collecting feeds every %s\n", timeBetweenRequests)

    // Step 2: Create a ticker
    ticker := time.NewTicker(timeBetweenRequests)
    defer ticker.Stop()

    // Step 3: Run immediately once, then on every tick
    for {
        if err := scrapeFeeds(s); err != nil {
            fmt.Fprintf(os.Stderr, "Error scraping feeds: %v\n", err)
        }

        // Wait for the next tick
        <-ticker.C
    }
}


func middlewareLoggedIn(
    handler func(s *state, cmd command, user database.User) error,
) func(*state, command) error {
    return func(s *state, cmd command) error {
        ctx := context.Background()

        // Load current user from config
        cfg := s.cfg
        user, err := s.db.GetUser(ctx, cfg.CurrentUserName)
        if err != nil {
            return fmt.Errorf("could not load current user: %w", err)
        }

        // Pass user to the wrapped handler
        return handler(s, cmd, user)
    }
}


func handlerAddFeed(s *state, cmd command, user database.User) error {
    if len(cmd.args) < 2 {
        return fmt.Errorf("usage: addfeed <name> <url>")
    }

    name := cmd.args[0]
    url := cmd.args[1]

    // Step 1: Create a new feed
    newFeed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
        ID:        uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        Name:      name,
        Url:       url,
        UserID:    user.ID,
    })
    if err != nil {
        return fmt.Errorf("could not create feed — maybe the URL already exists? %w", err)
    }

    // Step 2: Automatically create a feed follow
    _, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
        ID:        uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        UserID:    user.ID,
        FeedID:    newFeed.ID,
    })
    if err != nil {
        return fmt.Errorf("could not create feed follow: %w", err)
    }

    fmt.Printf("Feed created and followed successfully:\n%+v\n", newFeed)
    return nil
}



func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to fetch feeds:", err)
		os.Exit(1)
	}

	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	for _, f := range feeds {
		fmt.Printf("* %s\n  URL: %s\n  Added by: %s\n\n", f.Name, f.Url, f.UserName)
	}

	return nil
}

func handlerFollow(s *state, cmd command, user database.User) error {
    if len(cmd.args) < 1 {
        return fmt.Errorf("usage: follow <feed_url>")
    }

    feedURL := cmd.args[0]
    ctx := context.Background()

    // Find the feed by URL
    feed, err := s.db.GetFeedByUrl(ctx, feedURL)
    if err != nil {
        return fmt.Errorf("could not find feed with url %s: %w", feedURL, err)
    }

    // Step 1: Check if the user already follows this feed
    follows, err := s.db.GetFeedFollowsForUser(ctx, user.ID)
    if err != nil {
        return fmt.Errorf("could not get feed follows: %w", err)
    }
    for _, f := range follows {
        if f.FeedID == feed.ID {
            fmt.Printf("User %s already follows feed %s\n", user.Name, feed.Name)
            return nil
        }
    }

    // Step 2: Create feed follow
    _, err = s.db.CreateFeedFollow(ctx, database.CreateFeedFollowParams{
        ID:        uuid.New(),
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
        UserID:    user.ID,
        FeedID:    feed.ID,
    })
    if err != nil {
        // fallback for race conditions
        if strings.Contains(err.Error(), "unique_user_feed") {
            fmt.Printf("User %s already follows feed %s\n", user.Name, feed.Name)
            return nil
        }
        return fmt.Errorf("could not create feed follow: %w", err)
    }

    fmt.Printf("User %s is now following feed %s\n", user.Name, feed.Name)
    return nil
}



func handlerFollowing(s *state, cmd command, user database.User) error {
    ctx := context.Background()

    follows, err := s.db.GetFeedFollowsForUser(ctx, user.ID)
    if err != nil {
        return fmt.Errorf("could not get feed follows: %w", err)
    }

    if len(follows) == 0 {
        fmt.Println("You are not following any feeds yet.")
        return nil
    }

    fmt.Println("Feeds you are following:")
    for _, f := range follows {
        fmt.Printf("* %s\n", f.FeedName)
    }
    return nil
}
func handlerUnfollow(s *state, cmd command, user database.User) error {
    if len(cmd.args) < 1 {
        return fmt.Errorf("usage: unfollow <feed_url>")
    }

    feedURL := cmd.args[0]
    ctx := context.Background()

    // Find the feed by URL
    feed, err := s.db.GetFeedByUrl(ctx, feedURL)
    if err != nil {
        return fmt.Errorf("could not find feed with url %s: %w", feedURL, err)
    }

    // Delete the feed follow record using the struct param
    err = s.db.DeleteFeedFollow(ctx, database.DeleteFeedFollowParams{
        UserID: user.ID,
        FeedID: feed.ID,
    })
    if err != nil {
        return fmt.Errorf("could not unfollow feed: %w", err)
    }

    fmt.Printf("User %s has unfollowed feed %s\n", user.Name, feed.Name)
    return nil
}



func scrapeFeeds(s *state) error {
    ctx := context.Background()

    feed, err := s.db.GetNextFeedToFetch(ctx)
    if err != nil {
        return fmt.Errorf("could not get next feed: %w", err)
    }

    if err := s.db.MarkFeedFetched(ctx, feed.ID); err != nil {
        return fmt.Errorf("could not mark feed fetched: %w", err)
    }

    rssFeed, err := rss.FetchFeed(ctx, feed.Url)
    if err != nil {
        return fmt.Errorf("could not fetch feed %s: %w", feed.Url, err)
    }

    fmt.Printf("Fetched feed: %s\n", feed.Name)

    for _, item := range rssFeed.Channel.Item {
        pubTime := time.Now()
        if item.PubDate != "" {
            t, err := time.Parse(time.RFC1123Z, item.PubDate)
            if err != nil {
                t, err = time.Parse(time.RFC1123, item.PubDate)
                if err == nil {
                    pubTime = t
                }
            } else {
                pubTime = t
            }
        }

        err := s.db.CreatePost(ctx, database.CreatePostParams{
            ID:        uuid.New(),
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
            Title:     html.UnescapeString(item.Title),
            Url:       item.Link,
            Description: sql.NullString{
                String: html.UnescapeString(item.Description),
                Valid:  item.Description != "",
            },
            PublishedAt: sql.NullTime{
                Time:  pubTime,
                Valid: true,
            },
            FeedID: feed.ID,
        })

        if err != nil {
            if strings.Contains(err.Error(), "unique") {
                continue
            }
            fmt.Fprintf(os.Stderr, "Error inserting post %s: %v\n", item.Link, err)
        } else {
            fmt.Printf("- Saved post: %s\n", html.UnescapeString(item.Title))
        }
    }

    return nil
}


func handlerBrowse(s *state, cmd command, user database.User) error {
    ctx := context.Background()

    // Step 1: Determine limit
    limit := int32(2) // default
    if len(cmd.args) >= 1 {
        l, err := strconv.Atoi(cmd.args[0])
        if err != nil || l <= 0 {
            fmt.Fprintf(os.Stderr, "Invalid limit %q, using default %d\n", cmd.args[0], limit)
        } else {
            limit = int32(l)
        }
    }

    // Step 2: Get posts for the current user
    posts, err := s.db.GetPostsForUser(ctx, database.GetPostsForUserParams{
    	ID:    user.ID,
        Limit:  limit,
    })
    if err != nil {
        return fmt.Errorf("could not get posts for user %s: %w", user.Name, err)
    }

    // Step 3: Print posts
    if len(posts) == 0 {
        fmt.Println("No posts available.")
        return nil
    }

    fmt.Printf("Showing %d posts for user %s:\n", len(posts), user.Name)
    for _, p := range posts {
        published := "unknown"
        if p.PublishedAt.Valid {
            published = p.PublishedAt.Time.Format(time.RFC1123)
        }

        fmt.Printf("* %s (%s) - %s\n", p.Title, p.FeedName, published)

        if p.Description.Valid && p.Description.String != "" {
            fmt.Printf("  %s\n", p.Description.String)
        }

        fmt.Printf("  Link: %s\n\n", p.Url)
    }

    return nil
}




