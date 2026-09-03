package ocr

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"golang.org/x/time/rate"
)

// RateLimitConfig holds configuration for rate limiting and retries
type RateLimitConfig struct {
	// RequestInterval is the minimum interval between requests (e.g. 30m, 500ms, 2s).
	// If <= 0, no rate limiting is applied.
	RequestInterval time.Duration

	// RequestsPerMinute is kept for backwards compatibility / reporting
	RequestsPerMinute float64

	// MaxRetries is the maximum number of retry attempts
	// If 0 or negative, default retry count is applied
	MaxRetries int

	// BackoffMaxWait is the maximum wait time between retries
	// Defaults to 30 seconds if not specified
	BackoffMaxWait time.Duration
}

// ParseDuration parses a duration string, extending standard time.ParseDuration
// to support days ("d", "day", "days"), e.g. "1d", "2 days", "0.5d".
// Plain numbers without units are treated as seconds (e.g. "120" -> 120s).
func ParseDuration(input string) (time.Duration, error) {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return 0, nil
	}

	// Check if input is a plain number (int or float)
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		if val <= 0 {
			return 0, nil
		}
		return time.Duration(val * float64(time.Second)), nil
	}

	// Check for day units: e.g. "1d", "2days", "1.5 day"
	if strings.HasSuffix(s, "days") || strings.HasSuffix(s, "day") || strings.HasSuffix(s, "d") {
		numStr := s
		numStr = strings.TrimSuffix(numStr, "days")
		numStr = strings.TrimSuffix(numStr, "day")
		numStr = strings.TrimSuffix(numStr, "d")
		numStr = strings.TrimSpace(numStr)

		val, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration format %q: %w", input, err)
		}
		if val <= 0 {
			return 0, nil
		}
		return time.Duration(val * float64(24*time.Hour)), nil
	}

	// Fallback to standard Go time.ParseDuration (supports ns, us, ms, s, m, h)
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format %q: %w", input, err)
	}
	if d < 0 {
		return 0, nil
	}
	return d, nil
}

// ParseRateInterval parses a rate or interval string into a time.Duration interval between requests.
// Supported formats:
// - Interval durations: "500ms", "10s", "2m", "30m", "1h", "1d"
// - Rate expressions with '/': "2/h", "2/hour", "30/m", "30/min", "10/s", "100/d", "1/30m"
// - Plain numbers: "120" -> treated as seconds ("120s")
func ParseRateInterval(input string) (time.Duration, error) {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return 0, nil
	}

	// Handle rate expressions with '/' e.g. "2/h", "30/m", "10/s", "100/d", "1/30m"
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		countStr := strings.TrimSpace(parts[0])
		unitStr := strings.TrimSpace(parts[1])

		count, err := strconv.ParseFloat(countStr, 64)
		if err != nil || count <= 0 {
			return 0, fmt.Errorf("invalid rate count in %q: %w", input, err)
		}

		unitDuration, err := parseUnitDuration(unitStr)
		if err != nil {
			return 0, fmt.Errorf("invalid rate unit in %q: %w", input, err)
		}

		interval := time.Duration(float64(unitDuration) / count)
		return interval, nil
	}

	// Handle normalized unit names if someone typed e.g. "2min" or "2minutes" or "1hour"
	s = normalizeDurationUnits(s)

	return ParseDuration(s)
}

func parseUnitDuration(unitStr string) (time.Duration, error) {
	switch unitStr {
	case "s", "sec", "second", "seconds":
		return time.Second, nil
	case "m", "min", "minute", "minutes":
		return time.Minute, nil
	case "h", "hr", "hour", "hours":
		return time.Hour, nil
	case "d", "day", "days":
		return 24 * time.Hour, nil
	case "ms", "msec", "millisecond", "milliseconds":
		return time.Millisecond, nil
	default:
		return ParseDuration(unitStr)
	}
}

func normalizeDurationUnits(s string) string {
	if strings.HasSuffix(s, "milliseconds") {
		return strings.TrimSuffix(s, "milliseconds") + "ms"
	}
	if strings.HasSuffix(s, "millisecond") {
		return strings.TrimSuffix(s, "millisecond") + "ms"
	}
	if strings.HasSuffix(s, "msec") {
		return strings.TrimSuffix(s, "msec") + "ms"
	}
	if strings.HasSuffix(s, "minutes") {
		return strings.TrimSuffix(s, "minutes") + "m"
	}
	if strings.HasSuffix(s, "minute") {
		return strings.TrimSuffix(s, "minute") + "m"
	}
	if strings.HasSuffix(s, "mins") {
		return strings.TrimSuffix(s, "mins") + "m"
	}
	if strings.HasSuffix(s, "min") {
		return strings.TrimSuffix(s, "min") + "m"
	}
	if strings.HasSuffix(s, "hours") {
		return strings.TrimSuffix(s, "hours") + "h"
	}
	if strings.HasSuffix(s, "hour") {
		return strings.TrimSuffix(s, "hour") + "h"
	}
	if strings.HasSuffix(s, "hrs") {
		return strings.TrimSuffix(s, "hrs") + "h"
	}
	if strings.HasSuffix(s, "hr") {
		return strings.TrimSuffix(s, "hr") + "h"
	}
	if strings.HasSuffix(s, "seconds") {
		return strings.TrimSuffix(s, "seconds") + "s"
	}
	if strings.HasSuffix(s, "second") {
		return strings.TrimSuffix(s, "second") + "s"
	}
	if strings.HasSuffix(s, "secs") {
		return strings.TrimSuffix(s, "secs") + "s"
	}
	if strings.HasSuffix(s, "sec") {
		return strings.TrimSuffix(s, "sec") + "s"
	}
	return s
}

// RateLimitedLLM wraps an LLM client with rate limiting and retry capabilities
type RateLimitedLLM struct {
	llm          llms.Model
	rateLimiter  *rate.Limiter
	maxRetries   int
	backoffMin   time.Duration
	backoffMax   time.Duration
	backoffScale float64
}

// NewRateLimitedLLM creates a new rate-limited LLM client
func NewRateLimitedLLM(llm llms.Model, config RateLimitConfig) *RateLimitedLLM {
	var limiter *rate.Limiter
	interval := config.RequestInterval
	if interval <= 0 && config.RequestsPerMinute > 0 {
		interval = time.Duration(float64(time.Minute) / config.RequestsPerMinute)
	}

	if interval > 0 {
		limiter = rate.NewLimiter(rate.Every(interval), 1)
	}

	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	backoffMin := 1 * time.Second
	if interval > backoffMin {
		backoffMin = interval
	}

	backoffMax := config.BackoffMaxWait
	if backoffMax <= 0 {
		backoffMax = 30 * time.Second
	}
	if backoffMax < backoffMin {
		backoffMax = backoffMin
	}

	return &RateLimitedLLM{
		llm:          llm,
		rateLimiter:  limiter,
		maxRetries:   maxRetries,
		backoffMin:   backoffMin,
		backoffMax:   backoffMax,
		backoffScale: 2.0,
	}
}

// Call implements the llms.Model interface
func (r *RateLimitedLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if r.rateLimiter != nil {
		if err := r.rateLimiter.Wait(ctx); err != nil {
			return "", fmt.Errorf("rate limiter wait failed: %w", err)
		}
	}

	var lastErr error
	attempt := 0

	for {
		response, err := r.llm.Call(ctx, prompt, options...)
		if err == nil {
			return response, nil
		}

		if attempt >= r.maxRetries {
			if lastErr != nil {
				return "", fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
			}
			return "", err
		}

		backoff := r.backoffMin * time.Duration(1<<uint(attempt))
		if backoff > r.backoffMax {
			backoff = r.backoffMax
		}
		jitter := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(jitter):
			attempt++
			lastErr = err
		}
	}
}

// GenerateContent implements the llms.Model interface
func (r *RateLimitedLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if r.rateLimiter != nil {
		if err := r.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter wait failed: %w", err)
		}
	}

	var lastErr error
	attempt := 0

	for {
		resp, err := r.llm.GenerateContent(ctx, messages, options...)
		if err == nil {
			return resp, nil
		}

		if attempt >= r.maxRetries {
			if lastErr != nil {
				return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
			}
			return nil, err
		}

		backoff := r.backoffMin * time.Duration(1<<uint(attempt))
		if backoff > r.backoffMax {
			backoff = r.backoffMax
		}
		jitter := time.Duration(float64(backoff) * (0.8 + 0.4*rand.Float64()))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(jitter):
			attempt++
			lastErr = err
		}
	}
}
