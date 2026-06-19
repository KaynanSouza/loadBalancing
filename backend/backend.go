package backend

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)


var (
	ctx = context.Background()
	rdb = redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
)

type Backend struct {
	Addr           string
	URL            *url.URL
	alive          bool
	rProxy         *httputil.ReverseProxy
	mux            sync.RWMutex
	activeConn     int
	totalReq       int
	totalLatencyMs int
	weight         int
}

func New(addr string) *Backend {
	url, err := url.Parse(addr)
	if err != nil {
		log.Fatal(err)
	}

	return &Backend{
		Addr:           addr,
		URL:            url,
		alive:          true,
		activeConn:     0,
		totalReq:       0,
		totalLatencyMs: 0,
		weight:         0,
	}
}

func (b *Backend) Address() string {
	return b.Addr
}

func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.alive
	b.mux.RUnlock()
	return
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.alive = alive
	b.mux.Unlock()
}

func (b *Backend) SetReverseProxy(proxy *httputil.ReverseProxy) {
	b.rProxy = proxy
}

func (b *Backend) ReverseProxy() *httputil.ReverseProxy {
	return b.rProxy
}

func (b *Backend) Serve(rw http.ResponseWriter, req *http.Request) {
	b.activeConn += 1
	b.totalReq += 1
	rand.Seed(time.Now().UnixNano())
	ms := rand.Intn(250)
	time.Sleep(time.Duration(ms) * time.Millisecond)
	b.totalLatencyMs += ms
	fmt.Fprintf(rw, "(%s) Returned response in %d(ms)\n", b.Address(), ms)
	b.activeConn -= 1
}

func (b *Backend) ActiveConnections() int {
	return b.activeConn
}

func (b *Backend) TotalRequests() int {
	return b.totalReq
}

func (b *Backend) AverageLatency() int {
	if b.totalReq == 0 {
		return 75
	}
	return b.totalLatencyMs / b.totalReq
}

func (b *Backend) RequestHandler() func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		b.Serve(rw, req)
	}
}

func (b *Backend) Weight() int {
	return b.weight
}

func (b *Backend) SetWeight(weight int) {
	b.mux.Lock()
	b.weight = weight
	b.mux.Unlock()
}

func (b *Backend) IncConn() {
	b.mux.Lock()
	b.activeConn++
	b.totalReq++
	b.mux.Unlock()
}

func (b *Backend) DecConn(latencyMs int) {
	b.mux.Lock()
	b.activeConn--
	b.totalLatencyMs += latencyMs
	b.mux.Unlock()
}


func (b *Backend) GetPredictiveScore() float64 {
    key := "ai_score:" + b.Addr
    
    val, err := rdb.HGet(ctx, key, "score").Float64()
    if err != nil {
        return 50.0 
    }
    return val
}