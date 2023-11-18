package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/hulkholden/gowebgpu/static"
)

var (
	//go:embed templates/*
	templatesFS embed.FS
	indexTmpl   = template.Must(template.ParseFS(templatesFS, "templates/index.html"))

	port     = flag.Int("port", 80, "http port to listen on")
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

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), logRequest(http.DefaultServeMux)); err != nil {
		log.Println("Failed to start server", err)
		os.Exit(1)
	}
}
