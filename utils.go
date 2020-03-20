package main

import (
	"bytes"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

const httpClientTimeout = 15 * time.Second
const dialTimeout = 7 * time.Second

var httpClient = &fasthttp.Client{
	ReadTimeout:         30 * time.Second,
	MaxConnsPerHost:     233,
	MaxIdleConnDuration: 15 * time.Minute,
	ReadBufferSize:      1024 * 8,
	Dial: func(addr string) (net.Conn, error) {
		// no suitable address found => ipv6 can not dial to ipv4,..
		hostname, port, err := net.SplitHostPort(addr)
		if err != nil {
			if err1, ok := err.(*net.AddrError); ok && strings.Index(err1.Err, "missing port") != -1 {
				hostname, port, err = net.SplitHostPort(strings.TrimRight(addr, ":") + ":80")
			}
			if err != nil {
				return nil, err
			}
		}
		if port == "" || port == ":" {
			port = "80"
		}
		return fasthttp.DialDualStackTimeout("["+hostname+"]:"+port, dialTimeout)
	},
}

var errEncodingNotSupported = errors.New("response content encoding not supported")

func getResponseBody(resp *fasthttp.Response) ([]byte, error) {
	var contentEncoding = resp.Header.Peek("Content-Encoding")
	if len(contentEncoding) < 1 {
		return resp.Body(), nil
	}
	if bytes.Equal(contentEncoding, []byte("gzip")) {
		return resp.BodyGunzip()
	}
	if bytes.Equal(contentEncoding, []byte("deflate")) {
		return resp.BodyInflate()
	}
	return nil, errEncodingNotSupported
}
