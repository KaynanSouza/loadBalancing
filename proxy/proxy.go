package proxy

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/wfgilman/balancer/backend"
	"github.com/wfgilman/balancer/pool"
	"github.com/wfgilman/balancer/utils"
)

func New(targetAddr string) *httputil.ReverseProxy {
	// 1. Resolve o Bug do DNS: Usamos o Parse nativo para ler a URL corretamente
	targetUrl, err := url.Parse(targetAddr)
	if err != nil {
		log.Fatal("Erro fatal ao fazer o parse da URL de destino:", err)
	}
	
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)
	
	// 2. Proteção SRE: Intercepta a requisição para reescrever o cabeçalho
	// e evitar o erro "403 Forbidden" do Kubernetes
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = targetUrl.Host // Mente para o K8s que a requisição é interna
	}
	
	return proxy
}

func ErrorHandler(p *httputil.ReverseProxy, sp pool.Pool, be *backend.Backend) func(rw http.ResponseWriter, req *http.Request, e error) {
	return func(rw http.ResponseWriter, req *http.Request, e error) {
		log.Printf("[%s] Erro no Proxy: %v\n", be.Address(), e)
		
		// Marca o nó como morto imediatamente
		be.SetAlive(false)

		retries := utils.CountRetries(req)
		if retries >= 3 {
			log.Printf("[%s] Máximo de retentativas atingido. Desistindo.\n", req.RemoteAddr)
			http.Error(rw, "502 Bad Gateway - Falha na comunicação com o Pod", http.StatusBadGateway)
			return
		}

		attempts := utils.CountAttempts(req)
		log.Printf("[%s] Tentativa de roteamento %d\n", req.RemoteAddr, attempts)
		
		// Injeta os contadores no contexto e tenta o próximo servidor na pool
		ctx := context.WithValue(req.Context(), utils.Retries, retries+1)
		ctx = context.WithValue(ctx, utils.Attempts, attempts+1)
		sp.Serve(rw, req.WithContext(ctx))
	}
}
