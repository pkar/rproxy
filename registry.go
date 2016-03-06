package rproxy

import (
	"container/ring"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/pkar/log"
)

var (
	// ErrServiceNotFound for when there are no active upstream hosts.
	ErrServiceNotFound = errors.New("upstream host not found")
)

// Registry is an interface used to lookup the target upstream host
// for a given host.
type Registry interface {
	Add(host string, upstream []string) // Add an upstream hosts to the registry
	Enable(host, upstream string)       // Enable an upstream upstream in the registry
	Disable(host, upstream string)      // Disable an upstream upstream from the registry
	WaitPing(host, upstream string)     // Disable an upstream upstream from the registry
	Next(host string) (string, error)   // Return the upstream list for the given host
}

// Upstream is the the upstream upstream for the proxy.
type Upstream struct {
	URL      string
	StatusOK bool
}

// DefaultRegistry ...
type DefaultRegistry struct {
	Hosts map[string]*ring.Ring
	mu    sync.RWMutex
}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{Hosts: map[string]*ring.Ring{}}
}

// Add adds the given upstream upstream for the host.
func (r *DefaultRegistry) Add(host string, upstreams []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Hosts[host]; !ok {
		r.Hosts[host] = ring.New(len(upstreams))
	}
	for _, upstream := range upstreams {
		r.Hosts[host].Value = &Upstream{upstream, true}
		r.Hosts[host] = r.Hosts[host].Next()
		log.Info.Println("added", upstream, "for host", host)
	}
}

// Next returns an upstream for the given host. It ignores
// any upstream hosts that are disabled.
func (r *DefaultRegistry) Next(host string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	current, ok := r.Hosts[host]
	if !ok {
		log.Error.Printf("host %s not found in %v", host, r.Hosts)
		return "", ErrServiceNotFound
	}

	next := current
	for {
		next = next.Next()
		upstream := next.Value.(*Upstream)
		if upstream.StatusOK {
			r.Hosts[host] = next
			return upstream.URL, nil
		}
		if next == current {
			log.Error.Println("all upstream hosts disabled")
			return "", ErrServiceNotFound
		}
	}
}

func (r *DefaultRegistry) Find(host string, upstream string) *Upstream {
	r.mu.Lock()
	current, ok := r.Hosts[host]
	r.mu.Unlock()

	if !ok {
		return nil
	}
	next := current
	for {
		next = next.Next()
		u := next.Value.(*Upstream)
		if u.URL == upstream {
			return u
		}
		if next == current {
			return nil
		}
	}
}

// Enable an upstream hosts StatusOK
func (r *DefaultRegistry) Enable(host, upstream string) {
	u := r.Find(host, upstream)
	if u != nil {
		u.StatusOK = true
	}
}

// Disable disables the given upstream upstream for the host.
func (r *DefaultRegistry) Disable(host, upstream string) {
	u := r.Find(host, upstream)
	if u != nil {
		u.StatusOK = false
	}
}

// waitPing attempts to re-enable a down host by dialing
// it until it comes back on line at which point the upstream
// will be added back to the pool.
func (r *DefaultRegistry) WaitPing(host, upstream string) {
	multiplier := 2
	delay := 10
	nTries := 0
	for {
		upstream = strings.Replace(upstream, "http://", "", 1)
		upstream = strings.Replace(upstream, "https://", "", 1)
		conn, err := net.Dial("tcp", upstream)
		if conn != nil {
			conn.Close()
			break
		}
		if err != nil {
			nTries += 1
			log.Error.Println("attempt", nTries, upstream, "waiting", delay, "ms")
			time.Sleep(time.Duration(delay) * time.Millisecond)
			delay = delay * multiplier
			if delay > 40000 {
				delay = 1
			}
			continue
		}
	}
	log.Info.Println("enabling", upstream, "for host", host)
	r.Enable(host, upstream)
}
