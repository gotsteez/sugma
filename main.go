package main

import (
	"log"
	"net/http"
	"net/url"

	"net/http/httputil"

	// "net/url"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

func main() {
	r := router()

	log.Println("Starting http server")
	http.ListenAndServe(":8080", r)
}

func router() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		AllowedOrigins: []string{"*"},
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Get("/", test)
	r.Get("/cf", CfBypass)

	return r
}

func test(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rawurl := query.Get("url")
	redirect, err := url.Parse(rawurl)

	if err != nil || rawurl == "" {
		log.Println("Error ocurred while parsing url ", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	r.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/84.0.4147.105 Safari/537.36")
	proxy := httputil.NewSingleHostReverseProxy(redirect)
	proxy.ServeHTTP(w, r)
}

// CfBypass is handles your request, changes tls fingerprint, useragent, etc, and bypasses cf
func CfBypass(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rawurl := query.Get("url")
	redirect, err := url.Parse(rawurl)

	if err != nil || rawurl == "" {
		log.Println("Error ocurred while parsing url ", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	r.URL = redirect
	r.Proto = "HTTP/2.0"
	r.ProtoMajor = 2
	r.ProtoMinor = 0

	tr := newHTTPTransport(dialTLSFn)
	c := &http.Client{Transport: tr}
	resp, err := c.Get(rawurl)

	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	dumped, err := httputil.DumpResponse(resp, true)
	w.Write([]byte(dumped))

	// returns 403 error with blank body
	// proxy := httputil.NewSingleHostReverseProxy(redirect)
	// proxy.Transport = tr
	// proxy.ServeHTTP(w, r)
}
