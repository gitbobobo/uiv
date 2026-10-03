package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Token     string
	PublicURL string
	DataDir   string
	Addr      string
	MaxSize   int64
}

func FromEnv() (Config, error) {
	c := Config{
		Token:     os.Getenv("UIV_TOKEN"),
		PublicURL: strings.TrimRight(os.Getenv("UIV_PUBLIC_URL"), "/"),
		DataDir:   envOr("UIV_DATA_DIR", "/data"),
		Addr:      envOr("UIV_ADDR", ":8080"),
	}
	if c.Token == "" {
		return c, errors.New("UIV_TOKEN is required: set it to a long random string, e.g. `openssl rand -hex 32`")
	}
	if c.PublicURL != "" {
		u, err := url.Parse(c.PublicURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return c, fmt.Errorf("UIV_PUBLIC_URL must look like https://example.com, got %q", c.PublicURL)
		}
	}
	size, err := ParseSize(envOr("UIV_MAX_SIZE", "200MB"))
	if err != nil {
		return c, fmt.Errorf("UIV_MAX_SIZE: %w", err)
	}
	c.MaxSize = size
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ParseSize accepts plain bytes or a number with a B/KB/MB/GB suffix (1024-based).
func ParseSize(s string) (int64, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	units := []struct {
		suffix string
		mul    int64
	}{{"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"G", 1 << 30}, {"M", 1 << 20}, {"K", 1 << 10}, {"B", 1}}
	mul := int64(1)
	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			s = strings.TrimSpace(strings.TrimSuffix(s, u.suffix))
			mul = u.mul
			break
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return n * mul, nil
}
