package sona

import (
	"context"
	"github.com/cornelk/hashmap"
	"github.com/snowmerak/sona/lib/listmap"
	"golang.org/x/net/http2"
	"net/http"
)

type Sona struct {
	tlsFile struct {
		certFile string
		keyFile  string
	}
	connMap *hashmap.Map[string, *listmap.ListMap]
	server  *http.Server
}

func New() *Sona {
	return &Sona{
		connMap: hashmap.New[string, *listmap.ListMap](),
		server:  new(http.Server),
	}
}

func (s *Sona) SetAddr(addr string) *Sona {
	s.server.Addr = addr
	return s
}

func (s *Sona) SetTLSFile(certFile, keyFile string) *Sona {
	s.tlsFile.certFile = certFile
	s.tlsFile.keyFile = keyFile
	return s
}

func (s *Sona) Run() error {
	if s.server.Addr == "" {
		return errAddrEmpty
	}
	if err := http2.ConfigureServer(s.server, nil); err != nil {
		return err
	}
	if s.tlsFile.certFile != "" && s.tlsFile.keyFile != "" {
		return s.server.ListenAndServeTLS(s.tlsFile.certFile, s.tlsFile.keyFile)
	}
	return s.server.ListenAndServe()
}

func (s *Sona) Stop() error {
	return s.server.Close()
}

func (s *Sona) Setup(ctx context.Context) *Sona {
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

		value, _ := s.connMap.GetOrInsert(path, listmap.New())
		writeErrorOccurred := make(chan struct{}, 1)
		value.Append(path, func(data []byte) {
			data = append(data, '\n')
			if _, err := w.Write(data); err != nil {
				writeErrorOccurred <- struct{}{}
				return
			}
			flusher.Flush()
		})

		done := ctx.Done()

		select {
		case <-done:
		case <-writeErrorOccurred:
		}

		value.Remove(path)
	})

	s.server.Handler = mux

	return s
}

func (s *Sona) Broadcast(path string, data []byte) {
	value, ok := s.connMap.Get(path)
	if !ok {
		return
	}
	value.Iterate(data)
}
