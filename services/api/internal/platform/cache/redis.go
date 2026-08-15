package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient creates a new Redis client from a redis:// URL and verifies the
// connection with a ping.
func NewClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	if redisURL == "" {
		return nil, fmt.Errorf("redis url is empty")
	}

	opts, err := parseRedisURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}

	client := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

// Close gracefully closes the Redis client.
func Close(client *redis.Client) {
	if client != nil {
		_ = client.Close()
	}
}

func parseRedisURL(redisURL string) (*redis.Options, error) {
	u, err := url.Parse(redisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	opts := &redis.Options{}

	switch u.Scheme {
	case "redis", "rediss":
		// ok
	default:
		return nil, fmt.Errorf("unsupported redis scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return nil, fmt.Errorf("redis url missing host")
	}
	opts.Addr = u.Host

	if u.User != nil {
		password, ok := u.User.Password()
		if ok {
			opts.Password = password
		}
	}

	db := 0
	if u.Path != "" && u.Path != "/" {
		dbStr := u.Path[1:]
		db, err = strconv.Atoi(dbStr)
		if err != nil {
			return nil, fmt.Errorf("invalid redis database number: %s", dbStr)
		}
	}
	opts.DB = db

	if u.Scheme == "rediss" {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: hostWithoutPort(u.Host),
		}
	}

	return opts, nil
}

func hostWithoutPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}
