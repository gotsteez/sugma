package main

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"net/url"

	"net/http/httputil"

	// "net/url"

	"fmt"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
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
	requestHostName := redirect.Host
	w.Write([]byte("Requested url is " + requestHostName))
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

	roller, err := utls.NewRoller()
	if err != nil {
		log.Panicln("Error while creating new roller", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	requestHostName := redirect.Host

	log.Println("Dialing", requestHostName)
	conn, err := roller.Dial("tcp", requestHostName+":443", requestHostName)

	if err != nil {
		log.Println("Error occurred while dialing the server", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	resp, err := httpGetOverConn(conn, conn.HandshakeState.ServerHello.AlpnProtocol, redirect)

	if err != nil {
		w.Write([]byte("Error after httpGetOverConn " + err.Error()))
		return
	}

	defer resp.Body.Close()

	body, err := httputil.DumpResponse(resp, true)

	if err != nil {
		w.Write([]byte("Error while dumping request body " + err.Error()))
		return
	}

	w.Write(body)
}

func httpGetOverConn(conn net.Conn, alpn string, reqURL *url.URL) (*http.Response, error) {
	requestHostName := reqURL.Host
	req := &http.Request{
		Method: "GET",
		URL:    reqURL,
		Header: make(http.Header),
		Host:   requestHostName,
	}

	req.Header.Set("user-agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 12_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148")

	switch alpn {
	case "h2":
		req.Proto = "HTTP/2.0"
		req.ProtoMajor = 2
		req.ProtoMinor = 0

		tr := http2.Transport{}
		cConn, err := tr.NewClientConn(conn)

		defer tr.CloseIdleConnections()

		if err != nil {
			return nil, err
		}

		return cConn.RoundTrip(req)
	case "http/1,1":
		req.Proto = "HTTP/1.1"
		req.ProtoMajor = 1
		req.ProtoMinor = 1

		err := req.Write(conn)

		if err != nil {
			return nil, err
		}

		return http.ReadResponse(bufio.NewReader(conn), req)

	default:
		return nil, fmt.Errorf("unsupported ALPN: %v", alpn)
	}
}
