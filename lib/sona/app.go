package sona

import (
	"context"
	"github.com/cornelk/hashmap"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/snowmerak/sona/lib/listmap"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
	"net/http"
)

type Sona struct {
	tlsFile struct {
		certFile string
		keyFile  string
	}
	connMap   *hashmap.Map[string, *listmap.ListMap]
	sseServer *http.Server
	wsServer  *http.Server
}

func New() *Sona {
	return &Sona{
		connMap:   hashmap.New[string, *listmap.ListMap](),
		sseServer: nil,
		wsServer:  nil,
	}
}

func (s *Sona) SetTLSFile(certFile, keyFile string) *Sona {
	s.tlsFile.certFile = certFile
	s.tlsFile.keyFile = keyFile
	return s
}

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

func (s *Sona) EnableWS(ctx context.Context, addr string) *Sona {
	s.wsServer = new(http.Server)
	s.wsServer.Addr = addr

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		path := r.URL.Path
		name := r.RemoteAddr

		errorOccurred := make(chan struct{}, 1)
		value, _ := s.connMap.GetOrInsert(path, listmap.New())
		value.Append(name, func(data []byte) {
			defer func() {
				_ = recover()
			}()
			if err := wsutil.WriteServerBinary(conn, data); err != nil {
				errorOccurred <- struct{}{}
				close(errorOccurred)
				return
			}
		})
		defer value.Remove(name)

		done := ctx.Done()

		select {
		case <-done:
		case <-errorOccurred:
		}

		_ = conn.Close()
	})

	s.wsServer.Handler = mux

	return s
}

func (s *Sona) Run() error {
	if s.sseServer != nil {
		if err := http2.ConfigureServer(s.sseServer, nil); err != nil {
			return err
		}
	}
	if s.wsServer != nil {
		if err := http2.ConfigureServer(s.wsServer, nil); err != nil {
			return err
		}
	}

	eg := new(errgroup.Group)
	if s.tlsFile.certFile != "" && s.tlsFile.keyFile != "" {
		if s.sseServer != nil {
			eg.Go(func() error {
				if err := s.sseServer.ListenAndServeTLS(s.tlsFile.certFile, s.tlsFile.keyFile); err != nil {
					return err
				}
				return nil
			})
		}
		if s.wsServer != nil {
			eg.Go(func() error {
				if err := s.wsServer.ListenAndServeTLS(s.tlsFile.certFile, s.tlsFile.keyFile); err != nil {
					return err
				}
				return nil
			})
		}
	} else {
		if s.sseServer != nil {
			eg.Go(func() error {
				if err := s.sseServer.ListenAndServe(); err != nil {
					return err
				}
				return nil
			})
		}
		if s.wsServer != nil {
			eg.Go(func() error {
				if err := s.wsServer.ListenAndServe(); err != nil {
					return err
				}
				return nil
			})
		}
	}

	return eg.Wait()
}

func (s *Sona) Stop() {
	_ = s.sseServer.Close()
}

func (s *Sona) Broadcast(path string, data []byte) {
	value, ok := s.connMap.Get(path)
	if !ok {
		return
	}
	value.Iterate(data)
}
