# Balancer
A simple load balancer written in Go.

### Features
* [ ] Testing
* [x] Round Robin Method
* [x] Health Checks
* [x] Fewest Active Connections Method
* [x] Least Latency Method
* [x] Weighted Round Robin
* [x] IP Hash
* [x] Power of Two Choices (P2C)

### How it works
![Schematic](/assets/lb.png)

The application now acts as a load balancer for Kubernetes pods via the Kubernetes API Proxy.
On start-up, it registers a predefined list of Pod names and creates a backend representation for each, pointing to the local `kubectl proxy` endpoint (e.g. `http://127.0.0.1:8000/api/v1/namespaces/default/pods/...`).

Each backend is assigned a single host reverse proxy. The purpose of the proxy in this
application is to handle backend failure gracefully within the server pool. In Go, the
reverse proxy has an `ErrorHandler` method which can be customized to retry requests or
propagate issues with its assigned backend to the server pool.

Each backend is assigned to the server pool of the load balancer. The pool is responsible
for implementing the server rotation algorithms, including advanced methods like WRR (where the first pod gets a weight of 3, and the rest 1), IP Hash, and P2C. An HTTP server is created and implements a response handler for the pool that decides which backend to send the request to.

The `main` function runs a health check on an interval which makes a TCP connection to each backend
server to verify it is alive, then sets the status of each backend through the `Server` interface
accordingly.

### Usage
This application routes requests to Kubernetes pods. You must have a Kubernetes cluster running with the specific NGINX pods listed in `main.go` (or modify `main.go` to match your pods).

**1. Start the Kubernetes proxy:**
```bash
kubectl proxy --port=8000
```

**2. Start the Load Balancer:**
In another terminal, navigate to the root of the project and run:
```bash
go run . --port=8080 --algo=wrr
```

The application takes two flags:
```
--port    The port number on which to start the load balancer (default: 8000)
--algo    The balancing algorithm to use (default: alwaysfirst). Options are:
          "alwaysfirst"   Takes the first server in the slice
          "roundrobin"    Takes the next healthy server sequentially
          "leastlatency"  Takes the server with the lowest average response time
          "fewestconn"    Takes the server with the least active connections
          "iphash"        Routes the request based on the client's IP hash
          "p2c"           Power of Two Choices - picks two servers randomly and uses the least loaded
          "wrr"           Weighted Round Robin - rotates based on pre-assigned server weights
```

**3. Test the Load Balancer:**
In another window, run the following cURL command to see it in action:
```bash
curl http://localhost:8080
```

### Round Robin Algorithm
![Schematic](/assets/go-balancer.gif)

### Credits
The design of this application is inspired and informed by the content and code
in the online resources below.

* Load Balancer: https://kasvith.me/posts/lets-create-a-simple-lb-go/
* Load Balancer: https://betterprogramming.pub/building-a-load-balancer-in-go-3da3c7c46f30
* Graceful Server Shutdown: https://www.rudderstack.com/blog/implementing-graceful-shutdown-in-go
* Reverse Proxy: https://blog.joshsoftware.com/2021/05/25/simple-and-powerful-reverseproxy-in-go/
* Enums: https://www.sohamkamani.com/golang/enums/
