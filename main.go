package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"embed"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/hulkholden/gowebgpu/static"
)

var (
	//go:embed templates/*
	templatesFS embed.FS
	indexTmpl   = template.Must(template.ParseFS(templatesFS, "templates/index.html"))

	port     = flag.Int("port", 80, "http port to listen on")
	useTLS   = flag.Bool("tls", false, "enable HTTPS with a self-signed certificate")
	basePath = flag.String("base_path", "", "base path to serve on, e.g. '/foo/'")
)

type server struct {
	basePath string
}

func (s server) index(w http.ResponseWriter, r *http.Request) {
	// By default "/" matches any path - e.g. "/non-existent".
	// Is there a way to do this when the handler is registed?
	if r.URL.Path != s.basePath {
		// TODO: does returning 404 for "/" cause gce ingress to return 502s?
		if r.URL.Path != "/" {
			http.NotFound(w, r)
		}
		return
	}

	data := map[string]any{}
	indexTmpl.Execute(w, data)
}

// makeGzipHandler returns a HTTP HanderFunc which serves a gzipped version of the content.
func makeGzipHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		// TODO: figure this out from the underlying file if we use this for more than just the .wasm.
		w.Header().Set("Content-Type", "application/wasm")
		r.URL.Path += ".gz"
		r.URL.RawPath += ".gz"
		h.ServeHTTP(w, r)
	}
}

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sr := &statusRecorder{
			ResponseWriter: w,
			Status:         200,
		}
		handler.ServeHTTP(sr, r)
		log.Printf("%s %s %d %s\n", r.RemoteAddr, r.Method, sr.Status, r.URL)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	Status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.Status = status
	r.ResponseWriter.WriteHeader(status)
}

func canonicalizeBasePath(s string) string {
	bp := s
	if !strings.HasSuffix(bp, "/") {
		bp = bp + "/"
	}
	if !strings.HasPrefix(bp, "/") {
		bp = "/" + bp
	}
	return bp
}

func main() {
	flag.Parse()

	basePath := canonicalizeBasePath(*basePath)
	srv := server{
		basePath: basePath,
	}

	http.HandleFunc(basePath, srv.index)

	staticHandler := http.FileServer(http.FS(static.FS))
	http.Handle(basePath+"static/", http.StripPrefix(basePath+"static/", staticHandler))
	// If client.wasm is requested, redirect to a gzipped version.
	http.Handle(basePath+"static/client.wasm", http.StripPrefix(basePath+"static/", makeGzipHandler(staticHandler)))

	addr := fmt.Sprintf(":%d", *port)
	handler := logRequest(http.DefaultServeMux)

	if *useTLS {
		tlsCert, err := generateSelfSignedCert()
		if err != nil {
			log.Fatalf("Failed to generate self-signed certificate: %v", err)
		}
		srv := &http.Server{
			Addr:    addr,
			Handler: handler,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
			},
		}
		log.Printf("Listening on https://0.0.0.0%s", addr)
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			log.Println("Failed to start server", err)
			os.Exit(1)
		}
	} else {
		log.Printf("Listening on http://0.0.0.0%s", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Println("Failed to start server", err)
			os.Exit(1)
		}
	}
}

// generateSelfSignedCert creates an in-memory self-signed TLS certificate.
func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating key: %v", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generating serial number: %v", err)
	}

	tmpl := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{Organization: []string{"gowebgpu dev"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(0, 0, 0, 0), net.IPv6loopback},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("creating certificate: %v", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}
