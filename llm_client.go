package main

import (
	"paperless-gpt/ocr"
)

// RateLimitedLLM wraps an LLM client with rate limiting and retry capabilities
type RateLimitedLLM = ocr.RateLimitedLLM

// RateLimitConfig holds configuration for rate limiting and retries
type RateLimitConfig = ocr.RateLimitConfig

// NewRateLimitedLLM creates a new rate-limited LLM client
var NewRateLimitedLLM = ocr.NewRateLimitedLLM
