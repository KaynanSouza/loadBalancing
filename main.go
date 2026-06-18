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
		"nginx-nodes-784fdb7b89-269fh",
		"nginx-nodes-784fdb7b89-4r6bm",
		"nginx-nodes-784fdb7b89-5wc4d",
		"nginx-nodes-784fdb7b89-6sccb",
		"nginx-nodes-784fdb7b89-8xvxz",
		"nginx-nodes-784fdb7b89-gkgmk",
		"nginx-nodes-784fdb7b89-k6tnr",
		"nginx-nodes-784fdb7b89-kfzmm",
		"nginx-nodes-784fdb7b89-mkb4n",
		"nginx-nodes-784fdb7b89-zq4ql",
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
