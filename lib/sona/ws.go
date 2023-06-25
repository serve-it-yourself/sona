package sona

import (
	"context"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/snowmerak/sona/lib/listmap"
	"net/http"
)

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
