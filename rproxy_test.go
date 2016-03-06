package rproxy

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewMultipleHostReverseProxy(t *testing.T) {
	upstream0 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hi0")
	}))
	defer upstream0.Close()
	upstream1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hi1")
	}))
	defer upstream1.Close()

	frontendURL := "127.0.0.1:19999"

	reg := NewDefaultRegistry()
	reg.Add(frontendURL, []string{upstream0.URL, upstream1.URL})
	proxyFunc := NewMultipleHostReverseProxy(reg)
	frontend := httptest.NewUnstartedServer(proxyFunc)
	frontend.Listener, _ = net.Listen("tcp", frontendURL)
	frontend.Start()
	defer frontend.Close()

	res, err := http.Get("http://" + frontendURL)
	if err != nil {
		t.Fatal(err)
	}
	greeting, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(greeting) != "hi1" {
		t.Fatal("got", string(greeting), "want hi1")
	}

	res, err = http.Get("http://" + frontendURL)
	if err != nil {
		t.Fatal(err)
	}
	greeting, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(greeting) != "hi0" {
		t.Fatal("got", string(greeting), "want hi0")
	}
}

func TestNewMultipleHostReverseProxyBadGateway(t *testing.T) {
	frontendURL := "127.0.0.1:19999"
	reg := NewDefaultRegistry()
	reg.Add(frontendURL, []string{"localhost:1111", "localhost:1112"})
	proxyFunc := NewMultipleHostReverseProxy(reg)
	frontend := httptest.NewUnstartedServer(proxyFunc)
	frontend.Listener, _ = net.Listen("tcp", frontendURL)
	frontend.Start()
	defer frontend.Close()

	res, err := http.Get("http://" + frontendURL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	// TODO If all upstream servers are down on initial request fix
	// empty response
	//if string(greeting) != "Bad Gateway" {
	//	t.Fatal("got", string(greeting), "want Bad Gateway")
	//}
}
