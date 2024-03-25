package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/khainv198/go-asyncjob"
)

func main() {
	job2 := asyncjob.NewJob(func(ctx context.Context) error {
		return fmt.Errorf("my error job 1")
	}, &asyncjob.JobConfig{MaxRetry: 0, Name: "job 1"})

	job3 := asyncjob.NewJob(func(ctx context.Context) error {
		log.Print("i am job 2")
		return nil
	}, &asyncjob.JobConfig{MaxRetry: 1, Name: "job 2"})

	asyncjob.NewGroup(&asyncjob.JobGroupOptions{IsConcurrent: true}, job2, job3).Run()

	time.Sleep(10 * time.Second)
}
