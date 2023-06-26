# sona

sona is a simple broadcast server for sending messages to multiple clients.

## Installation

```bash
go get github.com/snowmerak/sona
```

## Example

```go
package main

import (
	"context"
	"github.com/snowmerak/sona/lib/sona"
	"log"
	"time"
)

func main() {
	ctx := context.Background()
	app := sona.New().EnableSSE(ctx, "0.0.0.0:8080").EnableWS(ctx, "0.0.0.0:8081")

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
```

This code is sending a message("hello, world") to all clients every second.

## Events

```go
package main

import (
	"context"
	"github.com/snowmerak/sona/lib/sona"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx := context.Background()
	app := sona.New().
		EnableSSE(ctx, "0.0.0.0:8080").
		EnableWS(ctx, "0.0.0.0:8081").
		OnConnect(func(w http.ResponseWriter, r *http.Request) {
			log.Println("connect")
		}).
		OnSend(func(w http.ResponseWriter, r *http.Request) {
			log.Println("send")
		}).
		OnDisconnect(func(w http.ResponseWriter, r *http.Request) {
			log.Println("disconnect")
		})

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
```

1. OnConnect: Triggered when a client connects to the server.
2. OnSend: Triggered when the server sends a message to the client.
3. OnDisconnect: Triggered when a client disconnects from the server.
