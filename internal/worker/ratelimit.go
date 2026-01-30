package worker

import (
	"context"

	"golang.org/x/time/rate"
)

// RateLimiters holds rate limiters for different targets
type RateLimiters struct {
	AWS *rate.Limiter
	CC  *rate.Limiter
	LLM *rate.Limiter
}

// NewRateLimiters creates rate limiters based on configuration
func NewRateLimiters(awsRate, ccRate, llmRate int) *RateLimiters {
	return &RateLimiters{
		AWS: rate.NewLimiter(rate.Limit(awsRate), 1),
		CC:  rate.NewLimiter(rate.Limit(ccRate), 1),
		LLM: rate.NewLimiter(rate.Limit(llmRate), 1),
	}
}

// WaitAWS waits for the AWS rate limiter
func (r *RateLimiters) WaitAWS(ctx context.Context) error {
	return r.AWS.Wait(ctx)
}

// WaitCC waits for the Confluent Cloud rate limiter
func (r *RateLimiters) WaitCC(ctx context.Context) error {
	return r.CC.Wait(ctx)
}

// WaitLLM waits for the LLM rate limiter
func (r *RateLimiters) WaitLLM(ctx context.Context) error {
	return r.LLM.Wait(ctx)
}
