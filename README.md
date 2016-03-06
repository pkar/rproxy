A host based reverse proxy

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/pkar/rproxy"
)

func main() {
	reg = rproxy.NewDefaultRegistry()
	reg.Add("localhost:9999", []string{"http://localhost:7777", "http://localhost:7778"})

	http.HandleFunc("/", rproxy.NewMultipleHostReverseProxy(reg))
	http.ListenAndServe("localhost:9999", nil)
}
```
