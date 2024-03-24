package main

import (
	"context"
	"log"

	"github.com/khainv198/go-asyncjob"
)

func main() {
	job := asyncjob.NewJob(func(ctx context.Context) error {
		log.Print("i am job 1")
		return nil
	}, &asyncjob.JobConfig{})

	asyncjob.NewGroup(&asyncjob.JobGroupOptions{}, job).BackgroundRun()
}
