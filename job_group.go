package asyncjob

import (
	"context"
	"sync"
	"time"
)

type jobGroup struct {
	jobs         []Job
	isConcurrent bool
	wg           *sync.WaitGroup
	maxTimeout   time.Duration
}

type JobGroupOptions struct {
	IsConcurrent bool
	MaxTimeout   time.Duration
}

type JonGroup interface {
	RunWithCtx(ctx context.Context) error
	Run() error
	RunWithCtxAndTimeout(ctx context.Context) error
	RunWithTimeout() error
}

func NewGroup(options *JobGroupOptions, jobs ...Job) JonGroup {
	g := &jobGroup{
		jobs:         jobs,
		isConcurrent: options.IsConcurrent,
		maxTimeout:   options.MaxTimeout,
		wg:           new(sync.WaitGroup),
	}

	if g.maxTimeout == 0 {
		g.maxTimeout = defaultMaxTimeout
	}

	return g
}

func (g *jobGroup) RunWithCtx(ctx context.Context) error {
	g.wg.Add(len(g.jobs))

	errChan := make(chan error, len(g.jobs))

	for i := range g.jobs {
		if !g.isConcurrent {
			err := g.runJobWithCtx(ctx, g.jobs[i])
			if err != nil {
				return err
			}

			errChan <- err
			g.wg.Done()

			continue
		}

		go func(j Job) {
			defer recovery()
			defer g.wg.Done()
			errChan <- g.runJobWithCtx(ctx, j)
		}(g.jobs[i])
	}

	g.wg.Wait()
	close(errChan)

	for i := 0; i < len(g.jobs); i++ {
		if v := <-errChan; v != nil {
			return v
		}
	}

	return nil
}

func (g *jobGroup) runJobWithCtx(ctx context.Context, j Job) error {
	err := j.ExecuteWithCtx(ctx)
	if err == nil {
		return nil
	}

	for {
		if j.GetSate() == StateRetryFailed {
			return err
		}

		if j.RetryWithCtx(ctx) == nil {
			return nil
		}
	}
}

func (g *jobGroup) Run() error {
	return g.RunWithCtx(context.Background())
}

func (g *jobGroup) RunWithCtxAndTimeout(ctx context.Context) error {
	gCtx, cancel := context.WithTimeout(ctx, g.maxTimeout)
	defer cancel()

	g.wg.Add(len(g.jobs))

	errChan := make(chan error, len(g.jobs))

	for i := range g.jobs {
		if !g.isConcurrent {
			err := g.runJobWithCtxAndTimeout(gCtx, g.jobs[i])
			if err != nil {
				return err
			}

			errChan <- err
			g.wg.Done()

			continue
		}

		go func(j Job) {
			defer recovery()
			defer g.wg.Done()
			errChan <- g.runJobWithCtxAndTimeout(gCtx, j)
		}(g.jobs[i])
	}

	g.wg.Wait()
	close(errChan)

	for i := 0; i < len(g.jobs); i++ {
		if v := <-errChan; v != nil {
			return v
		}
	}

	return nil
}

func (g *jobGroup) runJobWithCtxAndTimeout(ctx context.Context, j Job) error {
	if j.GetMaxTimeout() < g.maxTimeout {
		j.SetMaxTimeout(g.maxTimeout)
	}

	err := j.ExecuteWithCtxAndTimeout(ctx)
	if err == nil {
		return nil
	}

	for {
		if j.GetSate() == StateRetryFailed {
			return err
		}

		if j.RetryWithCtxAndTimeout(ctx) == nil {
			return nil
		}
	}
}

func (g *jobGroup) RunWithTimeout() error {
	return g.RunWithCtxAndTimeout(context.Background())
}
