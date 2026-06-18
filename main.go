package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/wfgilman/balancer/backend"
	"github.com/wfgilman/balancer/pool"
	"github.com/wfgilman/balancer/proxy"
)

func healthCheck() {
	t := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-t.C:
			log.Println("Health check started")
			serverPool.HealthCheck()
			log.Println("Health check complete")
		}
	}
}

func serverStats() {
	t := time.NewTicker(20 * time.Minute)
	for {
		select {
		case <-t.C:
			log.Println("Server Report...")
			serverPool.ServerStats()
		}
	}
}

var serverPool pool.Pool

func main() {
	port := flag.Int("port", 8080, "Enter port number for load balancer")
	algo := flag.String("algo", "alwaysfirst", "Balancing algorithm")
	flag.Parse()

	serverPool.Algorithm = *algo

	podNames := []string{
		"nginx-nodes-866bc755d6-26xbb",
		"nginx-nodes-866bc755d6-2c2p8",
		"nginx-nodes-866bc755d6-5ws7l",
		"nginx-nodes-866bc755d6-7vfq6",
		"nginx-nodes-866bc755d6-8nncn",
		"nginx-nodes-866bc755d6-99kws",
		"nginx-nodes-866bc755d6-dx42l",
		"nginx-nodes-866bc755d6-l6g2c",
		"nginx-nodes-866bc755d6-svxl4",
		"nginx-nodes-866bc755d6-x627s",
	}

	k8sProxyBase := "http://127.0.0.1:8080"

	for _, podName := range podNames {
		addr := fmt.Sprintf("%s/api/v1/namespaces/default/pods/%s:80/proxy/", k8sProxyBase, podName)

		be := backend.New(addr)

		if podName == podNames[0] {
			be.SetWeight(3)
		} else {
			be.SetWeight(1)
		}

		p := proxy.New(be.Address())
		p.ErrorHandler = proxy.ErrorHandler(p, serverPool, be)
		be.SetReverseProxy(p)

		serverPool.AddServer(be)
		log.Printf("Pod registrado na Pool: %s\n", podName)
	}

	// Create the load balancer server.
	server := http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", *port),
		Handler: http.HandlerFunc(serverPool.RequestHandler()),
	}

	go healthCheck()
	go serverStats()

	// Start the load balancer.
	log.Printf("Load Balancer K8s iniciado na porta %d com algoritmo: %s\\n", *port, *algo)
	server.ListenAndServe()
}
