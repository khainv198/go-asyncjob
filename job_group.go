package asyncjob

import (
	"context"
	"sync"
	"time"
)

type OnJobGroupRunSuccess func(context.Context)

type OnJobGroupRunError func(context.Context, error)

type jobGroup struct {
	jobs         []Job
	isConcurrent bool
	wg           *sync.WaitGroup
	maxTimeout   time.Duration
	onSuccess    OnJobGroupRunSuccess
	onError      OnJobGroupRunError
}

type JobGroupOptions struct {
	IsConcurrent bool
	OnSuccess    OnJobGroupRunSuccess
	OnError      OnJobGroupRunError
	MaxTimeout   time.Duration
}

type JonGroup interface {
	RunWithCtx(ctx context.Context) error
	Run() error
	RunWithCtxAndTimeout(ctx context.Context) error
	RunWithTimeout() error
	BackgroundRunWitCtx(ctx context.Context)
	BackgroundRun()
	BackgroundRunWitCtxAndTimeout(ctx context.Context)
	BackgroundRunWithTimeout()
}

func NewGroup(options *JobGroupOptions, jobs ...Job) JonGroup {
	g := &jobGroup{
		jobs:         jobs,
		isConcurrent: options.IsConcurrent,
		maxTimeout:   options.MaxTimeout,
		wg:           new(sync.WaitGroup),
		onSuccess:    options.OnSuccess,
		onError:      options.OnError,
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
				g.handleError(ctx, err)
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
			g.handleError(ctx, v)
			return v
		}
	}

	g.handleSuccess(ctx)

	return nil
}

func (g *jobGroup) handleError(ctx context.Context, err error) {
	if g.onError != nil {
		g.onError(ctx, err)
	}
}

func (g *jobGroup) handleSuccess(ctx context.Context) {
	if g.onSuccess != nil {
		g.onSuccess(ctx)
	}
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
				g.handleError(ctx, err)
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
			g.handleError(ctx, v)
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

func (g *jobGroup) BackgroundRunWitCtx(ctx context.Context) {
	var wg *sync.WaitGroup
	wg.Add(1)

	go func() {
		defer recovery()
		defer wg.Done()

		g.RunWithCtx(ctx)
	}()

	wg.Wait()
}

func (g *jobGroup) BackgroundRun() {
	g.BackgroundRunWitCtx(context.Background())
}

func (g *jobGroup) BackgroundRunWitCtxAndTimeout(ctx context.Context) {
	var wg *sync.WaitGroup
	wg.Add(1)

	go func() {
		defer recovery()
		defer wg.Done()

		g.RunWithCtxAndTimeout(ctx)
	}()

	wg.Wait()
}

func (g *jobGroup) BackgroundRunWithTimeout() {
	g.BackgroundRunWitCtxAndTimeout(context.Background())
}
