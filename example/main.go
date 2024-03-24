package main

import (
	"context"
	"log"
	"time"

	"github.com/khainv198/go-asyncjob"
)

func main() {
	job1 := asyncjob.NewJob(func(ctx context.Context) error {
		// log.Print("i am job 1")
		return nil
	}, nil)

	job2 := asyncjob.NewJob(func(ctx context.Context) error {
		log.Print("i am job 2")
		return nil
	}, nil)

	job3 := asyncjob.NewJob(func(ctx context.Context) error {
		log.Print("i am job 3")
		return nil
	}, nil)

	asyncjob.NewGroup(&asyncjob.JobGroupOptions{IsConcurrent: true}, job1, job2, job3).BackgroundRun()

	time.Sleep(10 * time.Second)
}
