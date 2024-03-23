package main

import (
	"context"
	"log"
	"time"

	"github.com/khainv198/go-asyncjob"
)

func main() {
	job1 := asyncjob.NewJob(func(ctx context.Context) error {
		time.Sleep(1 * time.Second)
		log.Print("i am job 1")
		return nil
	}, &asyncjob.JobConfig{})

	job2 := asyncjob.NewJob(func(ctx context.Context) error {
		time.Sleep(2 * time.Second)
		log.Print("i am job 2")
		return nil
	}, &asyncjob.JobConfig{})

	g := asyncjob.NewGroup(&asyncjob.JobGroupOptions{IsConcurrent: true}, job1, job2)
	g.Run()
}
