package rproxy

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	host     = "localhost:9999"
	upstream = []string{"localhost:7777", "localhost:7778"}
)

func TestNewDefaultRegistry(t *testing.T) {
	NewDefaultRegistry()
}

func TestAdd(t *testing.T) {
	reg := NewDefaultRegistry()
	reg.Add(host, upstream)
	if _, ok := reg.Hosts[host]; !ok {
		t.Fatal(host, "host not set")
	}
}

func TestFind(t *testing.T) {
	reg := NewDefaultRegistry()
	reg.Add(host, upstream)

	var findTests = []struct {
		host     string
		upstream string
		out      *Upstream
	}{
		{"localhost:9999", "localhost:7778", &Upstream{"localhost:7778", true}},
		{"localhost", "localhost:7778", nil},
		{"localhost:9999", "localhost:7771", nil},
	}

	for _, tt := range findTests {
		got := reg.Find(tt.host, tt.upstream)
		switch {
		case got == nil:
			if tt.out != nil {
				t.Errorf("Find(%s, %s) => %v, want %v", tt.host, tt.upstream, got, tt.out)
			}
		case tt.out.URL != got.URL:
			t.Errorf("Find(%s, %s) => %v, want %v", tt.host, tt.upstream, got, tt.out)
		}
	}
}

func TestNext(t *testing.T) {
	reg := NewDefaultRegistry()
	reg.Add(host, upstream)
	u0 := reg.Find(host, upstream[0])
	u1 := reg.Find(host, upstream[1])
	got1, _ := reg.Next(host)
	if got1 != u1.URL {
		t.Fatal("got", got1, "want", u1.URL)
	}
	got0, _ := reg.Next(host)
	if got0 != u0.URL {
		t.Fatal("got", got0, "want", u0.URL)
	}

	_, err := reg.Next("nothere")
	if err == nil {
		t.Fatal("expected err for host not there")
	}
	u0.StatusOK = false
	u1.StatusOK = false
	got, err := reg.Next(host)
	if err == nil {
		t.Fatal("expected err for upstream not there, got", got)
	}
}

func TestEnable(t *testing.T) {
	reg := NewDefaultRegistry()
	reg.Add(host, upstream)
	u := reg.Find(host, upstream[0])
	u.StatusOK = false
	reg.Enable(host, upstream[0])
	if u.StatusOK != true {
		t.Fatal("upstream host not enabled")
	}
}

func TestDisable(t *testing.T) {
	reg := NewDefaultRegistry()
	reg.Add(host, upstream)
	u := reg.Find(host, upstream[0])
	reg.Disable(host, upstream[0])
	if u.StatusOK != false {
		t.Fatal("upstream host not enabled")
	}
}

func TestWaitPing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hi")
	}))
	defer ts.Close()

	reg := NewDefaultRegistry()
	reg.Add(host, []string{ts.URL})
	reg.WaitPing(host, ts.URL)
}

func TestWaitPingUpstreamDown(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hi")
	}))
	defer ts.Close()
	url := "127.0.0.1:19999"

	reg := NewDefaultRegistry()
	reg.Add(host, []string{url})
	done := make(chan bool)
	go func() {
		reg.WaitPing(host, url)
		u := reg.Find(host, url)
		if u.StatusOK != true {
			t.Fatal("upstream host not enabled")
		}
		done <- true
	}()
	time.Sleep(time.Duration(50) * time.Millisecond)
	ts.Listener, _ = net.Listen("tcp", url)
	ts.Start()
	<-done
}
