package sona

import (
	"context"
	"github.com/snowmerak/sona/lib/listmap"
	"net/http"
)

func (s *Sona) EnableSSE(ctx context.Context, addr string) *Sona {
	s.sseServer = new(http.Server)
	s.sseServer.Addr = addr

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		path := r.URL.Path
		name := r.RemoteAddr

		value, _ := s.connMap.GetOrInsert(path, listmap.New())
		writeErrorOccurred := make(chan struct{}, 1)
		value.Append(name, func(data []byte) {
			defer func() {
				if err := recover(); err != nil {
					writeErrorOccurred <- struct{}{}
				}
			}()
			data = append(data, '\n')
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		})
		defer value.Remove(name)

		done := ctx.Done()

		select {
		case <-done:
		case <-writeErrorOccurred:
		}
	})

	s.sseServer.Handler = mux

	return s
}
