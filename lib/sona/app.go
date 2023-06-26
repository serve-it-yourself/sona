package sona

import (
	"github.com/cornelk/hashmap"
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

	connMap    *hashmap.Map[string, *listmap.ListMap]
	sseServer  *http.Server
	wsServer   *http.Server
	tcpHandler func(data []byte)

	onConnect    func(w http.ResponseWriter, r *http.Request)
	onSend       func(w http.ResponseWriter, r *http.Request)
	onDisconnect func(w http.ResponseWriter, r *http.Request)
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
