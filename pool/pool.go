package pool

import (
	"errors"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/wfgilman/balancer/server"
)

const (
	AlwaysFirst        string = "alwaysfirst"
	RoundRobin                = "roundrobin"
	LeastLatency              = "leastlatency"
	FewestConn                = "fewestconn" // Least Cpnnections
	IPHash                    = "iphash"
	PowerOfTwoChoices         = "p2c"
	WeightedRoundRobin        = "wrr"
)

type Pool struct {
	Current   uint64
	Servers   []server.Server
	Algorithm string
}

func (p *Pool) AddServer(server server.Server) {
	p.Servers = append(p.Servers, server)
}

func (p *Pool) RequestHandler() func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		p.Serve(rw, req)
	}
}

func (p *Pool) Serve(rw http.ResponseWriter, req *http.Request) {
	server := p.GetNextServer(req)
	proxy := server.ReverseProxy()

	req.Host = server.Address()

	proxy.ServeHTTP(rw, req)
}

func (p *Pool) GetNextServer(req *http.Request) server.Server {
	switch p.Algorithm {
	case AlwaysFirst:
		return p.Servers[0]
	case RoundRobin:
		nextIndex := int(atomic.AddUint64(&p.Current, uint64(1)) % uint64(len(p.Servers)))
		length := len(p.Servers) + nextIndex
		for i := nextIndex; i < length; i++ {
			index := i % len(p.Servers)
			if p.Servers[index].IsAlive() {
				if i != nextIndex {
					atomic.StoreUint64(&p.Current, uint64(index))
				}
				return p.Servers[index]
			}
		}
		panic("No healthy backends exist")
	case LeastLatency:
		var s server.Server
		min := math.MaxInt
		for _, server := range p.Servers {
			latency := server.AverageLatency()
			if server.IsAlive() && latency < min {
				min = latency
				s = server
			}
		}
		return s
	case FewestConn:
		var s server.Server
		min := math.MaxInt
		for _, server := range p.Servers {
			activeConn := server.ActiveConnections()
			if server.IsAlive() && activeConn < min {
				min = activeConn
				s = server
			}
		}
		if s != nil {
			return s
		}

	case IPHash:
		ip, _, err := net.SplitHostPort(req.RemoteAddr)
		if err != nil {
			ip = req.RemoteAddr
		}

		hash := fnv.New32a()
		hash.Write([]byte(ip))
		idx := int(hash.Sum32() % uint32(len(p.Servers)))

		if !p.Servers[idx].IsAlive() {
			return p.Servers[(idx+1)%len(p.Servers)]
		}

		return p.Servers[idx]

	case PowerOfTwoChoices:
		rand.Seed(time.Now().UnixNano())

		idx1, idx2 := rand.Intn(len(p.Servers)), rand.Intn(len(p.Servers))
		srv1, srv2 := p.Servers[idx1], p.Servers[idx2]

		if srv1.IsAlive() && (!srv2.IsAlive() || srv1.ActiveConnections() <= srv2.ActiveConnections()) {
			return srv1
		}

		if srv2.IsAlive() {
			return srv2
		}

	case WeightedRoundRobin:
		totalWeight := 0
		for _, s := range p.Servers {
			if s.IsAlive() {
				totalWeight += s.Weight()
			}
		}
		if totalWeight == 0 {
			break // Fallback se todos os pesos forem 0
		}

		rand.Seed(time.Now().UnixNano())
		randomPoint := rand.Intn(totalWeight)
		currentSum := 0

		for _, s := range p.Servers {
			if s.IsAlive() {
				currentSum += s.Weight()
				if randomPoint < currentSum {
					return s
				}
			}
		}
	}

	for _, s := range p.Servers {
		if s.IsAlive() {
			return s
		}
	}
	panic("No backends exist")
}

func (p *Pool) GetServer(targetAddr string) (server.Server, error) {
	for _, server := range p.Servers {
		if server.Address() == targetAddr {
			return server, nil
		}
	}
	return nil, errors.New("Server not found")
}

func (p *Pool) HealthCheck() {
	for _, server := range p.Servers {
		if server.IsAlive() {
			alive := isServerAlive(server)
			server.SetAlive(alive)
			if !alive {
				log.Printf("(%s) is down\n", server.Address())
			}
		}
	}
}

func isServerAlive(server server.Server) bool {
	url := &url.URL{
		Scheme: "http",
		Host:   server.Address(),
	}
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", url.Host, timeout)
	if err != nil {
		log.Printf("[%s] unreachable, error: ", server.Address(), err)
		return false
	}
	defer conn.Close()
	return true
}

func (p *Pool) ServerStats() {
	for _, s := range p.Servers {
		log.Printf("(%s) Alive %v, Active %d, Total %d, Latency %d(ms)\n", s.Address(), s.IsAlive(), s.ActiveConnections(), s.TotalRequests(), s.AverageLatency())
	}
}
