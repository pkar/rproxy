package rproxy

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/pkar/log"
)

// Common errors
var (
	ErrInvalidService = errors.New("invalid service")
)

// DialWithFailover is a dialer which on failed connections
// disables upstream hosts and randomly attempts to connect to another.
func DialWithFailover(protocol, host, upstream string, reg Registry) (conn net.Conn, err error) {
	for {

		upstream = strings.Replace(upstream, "http://", "", 1)
		upstream = strings.Replace(upstream, "https://", "", 1)

		// Try to connect
		conn, err = net.Dial(protocol, upstream)
		if err == nil {
			break
		}

		log.Error.Println(err)
		reg.Disable(host, upstream)
		go reg.WaitPing(host, upstream)
		upstream, err = reg.Next(host)
		if err != nil {
			return nil, fmt.Errorf("No upstream available for %s", host)
		}
	}
	// Success: return the connection.
	return conn, nil
}

// NewMultipleHostReverseProxy creates a reverse proxy handler
// initialized from a given registry.
func NewMultipleHostReverseProxy(reg Registry) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		host := req.Host
		upstream, err := reg.Next(host)
		if err != nil {
			log.Error.Println(err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}
		(&httputil.ReverseProxy{
			Director: func(r *http.Request) {
				r.URL.Scheme = "http"
				r.Host = host
				r.URL.Host = upstream
			},
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: func(protocol, h string) (net.Conn, error) {
					return DialWithFailover(protocol, host, upstream, reg)
				},
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}).ServeHTTP(w, req)
	}
}
