package sona

import "net/http"

func (s *Sona) OnConnect(action func(w http.ResponseWriter, r *http.Request)) *Sona {
	s.onConnect = action
	return s
}

func (s *Sona) OnSend(action func(w http.ResponseWriter, r *http.Request)) *Sona {
	s.onSend = action
	return s
}

func (s *Sona) OnDisconnect(action func(w http.ResponseWriter, r *http.Request)) *Sona {
	s.onDisconnect = action
	return s
}
