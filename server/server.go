package server

import (
	"net/http"
	"net/http/httputil"
)

type Server interface {
	Address() string
	IsAlive() bool
	SetAlive(alive bool)
	SetReverseProxy(*httputil.ReverseProxy)
	ReverseProxy() *httputil.ReverseProxy
	ActiveConnections() int
	TotalRequests() int
	AverageLatency() int
	Weight() int
	SetWeight(w int)
	Serve(rw http.ResponseWriter, req *http.Request)
	IncConn()
	DecConn(latencyMs int)
}
