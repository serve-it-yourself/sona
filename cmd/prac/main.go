package main

import (
	"context"
	"github.com/snowmerak/sona/lib/sona"
	"log"
	"time"
)

func main() {
	ctx := context.Background()
	app := sona.New().SetAddr("0.0.0.0:8080").Setup(ctx)

	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			app.Broadcast("/hello", []byte("hello world"))
			log.Println("broadcast")
		}
	}()

	if err := app.Run(); err != nil {
		panic(err)
	}
}
