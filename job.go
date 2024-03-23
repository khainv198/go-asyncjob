package asyncjob

import (
	"context"
	"time"
)

const (
	defaultMaxRetry                 = 3
	defaultMaxTimeout time.Duration = 30 * time.Second
	defaultDelay      time.Duration = 5 * time.Second
)

type JobHandler func(ctx context.Context) error

type JobState int8

const (
	StateInit JobState = 1 + iota
	StateRunning
	StateFailed
	StateTimeout
	StateCompleted
	StateRetryFailed
)

type JobConfig struct {
	Name       string
	MaxTimeout time.Duration
	MaxRetry   int
	Delay      time.Duration
}

type job struct {
	name       string
	handler    JobHandler
	state      JobState
	retryIndex int
	maxRetry   int
	maxTimeout time.Duration
	delay      time.Duration
}

type Job interface {
	ExecuteWithCtx(ctx context.Context) error
	ExecuteWithCtxAndTimeout(ctx context.Context) error
	Execute() error
	ExecuteWithTimeout() error
	RetryWithCtx(ctx context.Context) error
	RetryWithCtxAndTimeout(ctx context.Context) error
	Retry() error
	RetryWithTimeout() error
	GetSate() JobState
	GetMaxTimeout() time.Duration
	SetMaxTimeout(maxTimeout time.Duration)
	SetMaxRetry(retry int)
	SerDelay(delay time.Duration)
}

func NewJob(handler JobHandler, cfg *JobConfig) Job {
	j := &job{
		name:       cfg.Name,
		handler:    handler,
		state:      StateInit,
		retryIndex: -1,
		maxTimeout: cfg.MaxTimeout,
		maxRetry:   cfg.MaxRetry,
		delay:      cfg.Delay,
	}

	if j.maxTimeout == 0 {
		j.maxTimeout = defaultMaxTimeout
	}

	if j.maxRetry == 0 {
		j.maxRetry = defaultMaxRetry
	}

	if j.delay == 0 {
		j.delay = defaultDelay
	}

	return j
}

func (j *job) ExecuteWithCtx(ctx context.Context) error {
	j.state = StateRunning

	err := j.handler(ctx)
	if err != nil {
		j.state = StateFailed
		return err
	}

	j.state = StateCompleted
	return nil
}

func (j *job) ExecuteWithCtxAndTimeout(ctx context.Context) error {
	jobCtx, cancel := context.WithTimeout(ctx, j.maxTimeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		defer recovery()
		err := j.ExecuteWithCtx(jobCtx)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-jobCtx.Done():
		j.state = StateTimeout
		return jobCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (j *job) Execute() error {
	return j.ExecuteWithCtx(context.Background())
}

func (j *job) ExecuteWithTimeout() error {
	return j.ExecuteWithCtxAndTimeout(context.Background())
}

func (j *job) RetryWithCtx(ctx context.Context) error {
	if j.retryIndex >= j.maxRetry {
		j.state = StateRetryFailed
		return nil
	}

	j.retryIndex += 1
	time.Sleep(j.delay)

	err := j.ExecuteWithCtx(ctx)
	if err == nil {
		j.state = StateCompleted
		return err
	}

	if j.retryIndex == j.maxRetry {
		j.state = StateRetryFailed
	} else {
		j.state = StateFailed
	}

	return err
}

func (j *job) RetryWithCtxAndTimeout(ctx context.Context) error {
	jobCtx, cancel := context.WithTimeout(ctx, j.maxTimeout)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		defer recovery()
		err := j.RetryWithCtx(jobCtx)
		errChan <- err
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return err
		}
	case <-jobCtx.Done():
		j.state = StateTimeout
		return jobCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (j *job) Retry() error {
	return j.ExecuteWithCtx(context.Background())
}

func (j *job) RetryWithTimeout() error {
	return j.RetryWithCtxAndTimeout(context.Background())
}

func (j *job) GetSate() JobState {
	return j.state
}

func (j *job) GetMaxTimeout() time.Duration {
	return j.maxTimeout
}

func (j *job) SetMaxTimeout(maxTimeout time.Duration) {
	j.maxTimeout = maxTimeout
}

func (j *job) SetMaxRetry(retry int) {
	j.maxRetry = retry
}

func (j *job) SerDelay(delay time.Duration) {
	j.delay = delay
}
