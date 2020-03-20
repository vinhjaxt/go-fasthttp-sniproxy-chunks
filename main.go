package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"regexp"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/valyala/fasthttp"
)

var domainNameRegex = regexp.MustCompile(`^([a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*[\._]?$`)
var json = jsoniter.ConfigCompatibleWithStandardLibrary

const helloReadSize = 1024

var uncatchRecover = func() {
	if r := recover(); r != nil {
		log.Println("Uncatched error:", r, string(debug.Stack()))
	}
}

func httpsHandler(ctx *fasthttp.RequestCtx, hostname string, tcpAddr *net.TCPAddr) error {
	if ctx.Hijacked() {
		return nil
	}

	ioServer, err := net.DialTCP("tcp4", nil, tcpAddr)
	if err != nil {
		return err
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("Keep-Alive", "timeout=120, max=5")
	var hostnameBytes = []byte(hostname)
	ctx.Hijack(func(ioClient net.Conn) {
		if mustProxify(hostname) {
			buf := make([]byte, helloReadSize)
			var req []byte
			for {
				n, err := ioClient.Read(buf)
				if err != nil {
					log.Println(err)
					return
				}
				next := buf[:n]
				req = append(req, next...)
				hostnameIdx := bytes.Index(next, hostnameBytes) + 1
				isBreak := bytes.Index(req, hostnameBytes) != -1
				if hostnameIdx != 0 {
					ioServer.Write(next[0:hostnameIdx])
					ioServer.Write(next[hostnameIdx:])
					req = nil
					buf = nil
					break
				} else {
					ioServer.Write(next)
				}
				if isBreak {
					req = nil
					buf = nil
					break
				}
			}
		}
		go ioTransfer(ioClient, ioServer)
		ioTransfer(ioServer, ioClient)
	})
	return nil
}

func ioTransfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	defer uncatchRecover()
	_, err := io.Copy(destination, source)
	if err != nil {
		if err != io.EOF {
			// log.Println("ioTransfer", err)
		}
	}
}

var cacheAddrMapLock sync.RWMutex
var cacheTCPAddrMap = map[string]*net.TCPAddr{}

func requestHandler(ctx *fasthttp.RequestCtx) {
	defer uncatchRecover()
	// Some library must set header: Connection: keep-alive
	// ctx.Response.Header.Del("Connection")
	// ctx.Response.ConnectionClose() // ==> false

	// log.Println(string(ctx.Path()), string(ctx.Host()), ctx.String(), "\r\n\r\n", ctx.Request.String())

	host := string(ctx.Host())
	if len(host) < 1 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		log.Println("Reject: Empty host")
		return
	}

	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		if err1, ok := err.(*net.AddrError); ok && strings.Index(err1.Err, "missing port") != -1 {
			hostname, port, err = net.SplitHostPort(host + ":80")
		}
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			log.Println("Reject: Invalid host", host, err)
			return
		}
	}

	cacheAddrMapLock.RLock()
	tcpAddr, ok := cacheTCPAddrMap[host]
	cacheAddrMapLock.RUnlock()
	if ok == false {
		tcpAddr, err = getUsableIP(hostname, port)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			log.Println("No usable IP:", host, err)
			return
		}
		cacheAddrMapLock.Lock()
		cacheTCPAddrMap[host] = tcpAddr
		cacheAddrMapLock.Unlock()
	}

	// https connecttion
	if bytes.Equal(ctx.Method(), []byte("CONNECT")) {
		err = httpsHandler(ctx, hostname, tcpAddr)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			log.Println("httpsHandler:", host, err)
		}
		return
	}

	err = httpClient.DoTimeout(&ctx.Request, &ctx.Response, httpClientTimeout)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		log.Println("httpHandler:", host, err)
	}
}

var config struct {
	port             int
	domainList       string
	domainRegexList  string
	dnsEndpoint      string
	skipDNSTLSVerify string
}

var domainProxiesCache = map[string]bool{}
var domainProxiesCacheLock sync.RWMutex
var domainsRegex []*regexp.Regexp
var lineRegex = regexp.MustCompile(`[\r\n]+`)

func parseDomains() bool {
	if len(config.domainList) > 0 {
		c, err := ioutil.ReadFile(config.domainList)
		if err == nil {
			lines := lineRegex.Split(string(c), -1)
			for _, line := range lines {
				line = strings.Trim(line, "\r\n\t ")
				if len(line) < 1 || line[0] == '#' {
					continue
				}
				domainProxiesCacheLock.Lock()
				domainProxiesCache[line] = true
				domainProxiesCacheLock.Unlock()
			}
		} else {
			log.Println(err)
		}
	}
	if len(config.domainRegexList) > 0 {
		c, err := ioutil.ReadFile(config.domainRegexList)
		if err == nil {
			lines := lineRegex.Split(string(c), -1)
			for _, line := range lines {
				line = strings.Trim(line, "\r\n\t ")
				if len(line) < 1 || line[0] == '#' {
					continue
				}
				domainsRegex = append(domainsRegex, regexp.MustCompile(line))
			}
		} else {
			log.Println(err)
		}
	}
	if len(domainsRegex) < 1 && len(domainProxiesCache) < 1 {
		log.Println("No domains to proxy? Please specify a domain name in", config.domainList, "or", config.domainRegexList)
		return false
	}
	return true
}

func mustProxify(hostname string) bool {
	domainProxiesCacheLock.RLock()
	b, ok := domainProxiesCache[hostname]
	domainProxiesCacheLock.RUnlock()
	if ok {
		return b
	}
	b = false
	for _, re := range domainsRegex {
		b = re.MatchString(hostname)
		if b {
			break
		}
	}
	domainProxiesCacheLock.Lock()
	domainProxiesCache[hostname] = b
	domainProxiesCacheLock.Unlock()
	log.Println("Proxify:", hostname, b)
	return b
}

func main() {
	flag.IntVar(&config.port, "p", 8080, "listen port")
	flag.StringVar(&config.domainList, "d", "domains.txt", "Domains List File")
	flag.StringVar(&config.domainRegexList, "r", "domains-regex.txt", "Domains Regex List File")
	flag.StringVar(&config.dnsEndpoint, "dns", "https://1.0.0.1/dns-query", "DNS https endpoint")
	if runtime.GOOS == "windows" {
		// Ohhhh, fuck windows :'(, yeah I don't add roots ca
		flag.StringVar(&config.skipDNSTLSVerify, "skip-dns-tls-verify", "yes", "Skip verify DNS https endpoint")
	} else {
		flag.StringVar(&config.skipDNSTLSVerify, "skip-dns-tls-verify", "no", "Skip verify DNS https endpoint")
	}
	flag.Parse()
	dnsEndpointQs = config.dnsEndpoint + "?ct=application/dns-json&type=A&do=false&cd=false"
	log.Println("Config", config)

	config.skipDNSTLSVerify = strings.ToLower(config.skipDNSTLSVerify)
	if config.skipDNSTLSVerify == "yes" || config.skipDNSTLSVerify == "true" {
		httpClient.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	if parseDomains() == false {
		return
	}

	Server := &fasthttp.Server{
		Handler:              requestHandler,
		Name:                 "go-sniproxy",
		ReadTimeout:          10 * time.Second, // 120s
		WriteTimeout:         10 * time.Second,
		MaxKeepaliveDuration: 10 * time.Second,
		MaxRequestBodySize:   2 * 1024 * 1024, // 2MB
		DisableKeepalive:     false,
	}

	log.Println("Server running on:", config.port)
	if err := Server.ListenAndServe(":" + strconv.Itoa(config.port)); err != nil {
		log.Print("HTTP serve error:", err)
	}
}
