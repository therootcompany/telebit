package http01proxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// ListenAndServe will start the HTTP-01 proxy
func ListenAndServe(target string, timeout time.Duration) error {
	target = strings.TrimSuffix(target, "/")

	// TODO accept listener?
	targetURL, err := url.Parse(target)
	if nil != err {
		panic(err)
	}

	//proxyHandler := httputil.NewSingleHostReverseProxy(targetURL)
	proxyHandler := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Del("X-Forwarded-For")
			req.Header.Del("X-Forwarded-Proto")
			req.Header.Del("X-Forwarded-Port")
			req.Header.Del("X-Forwarded-Host")

			// We want the incoming host header to remain unchanged,
			// which is the domain name that is being challenged
			log.Printf("[debug] Incoming Host: %q", req.Header.Get("Host"))
			req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))

			targetQuery := targetURL.RawQuery
			req.URL.Scheme = targetURL.Scheme
			// But we want the proxy target to be updated to the new target
			req.URL.Host = targetURL.Host
			//req.Host = targetURL.Host
			req.URL.Path, req.URL.RawPath = joinURLPath(targetURL, req.URL)

			log.Printf("[debug] Target Host: %q", req.URL.Host)
			log.Printf("[debug] Target Path: %q", req.URL.Path)
			log.Printf("[debug] Target RawPath: %q", req.URL.Path)

			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}

			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}
		},
	}

	return http.ListenAndServe(":80", proxyHandler)
}

// Taken from https://golang.org/src/net/http/httputil/reverseproxy.go
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
