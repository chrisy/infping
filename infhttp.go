// infhttp.go HTTP probes
// Copyright: 2016 Tor Hveem
// Copyright: 2018 Chris Luke
// License: MIT

package main

import (
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pelletier/go-toml"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

func makeClient(family string, addrstr string) *http.Client {
	// Create a transport like http.DefaultTransport, but with a specified localAddr

	var addr net.Addr

	if len(addrstr) > 0 {
		addr, _ = net.ResolveIPAddr(family, addrstr)
	}

	dialer := (&net.Dialer{
		LocalAddr: addr,
		DualStack: false,
	}).Dial

	transport := &http.Transport{
		//Proxy: http.ProxyFromEnvironment,
		Dial: dialer,
		//MaxIdleConns:          100,
		//IdleConnTimeout:       90 * time.Second,
		//TLSHandshakeTimeout:   10 * time.Second,
		//ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
	}

	return client
}

func readHTTPPoints(config *toml.Tree, con client.Client) {
	verbose := config.Get("core.verbose").(bool)
	urls := config.Get("http.urls").([]interface{})
	ipv4_srcaddr := config.Get("http.ipv4_srcaddr").(string)
	ipv6_srcaddr := config.Get("http.ipv6_srcaddr").(string)

	if verbose {
		log.Printf("Going to fetch the following urls: %q", urls)
	}

	ipv4_client := makeClient("tcp4", ipv4_srcaddr)
	ipv6_client := makeClient("tcp6", ipv6_srcaddr)

	for {
		for _, v := range urls {
			url, _ := v.(string)
			go func(url string) {
				start := time.Now()

				var response *http.Response
				var err error

				if strings.HasPrefix(url, "ipv4:") {
					response, err = ipv4_client.Get(url[5:])
				} else if strings.HasPrefix(url, "ipv6:") {
					response, err = ipv6_client.Get(url[5:])
				} else {
					response, err = http.Get(url)
				}

				perr(err)
				contents, err := ioutil.ReadAll(response.Body)
				defer response.Body.Close()
				perr(err)

				elapsed := time.Since(start).Seconds()
				code := response.StatusCode
				bytes := len(contents)

				if verbose {
					log.Printf("http Url:%s, code: %s, bytes: %s, elapsed: %s",
						url, code, bytes, elapsed)
				}

				writeHTTPPoints(config, con, url, code, bytes, elapsed)
			}(url)
		}
		time.Sleep(time.Second * 30)
	}
}

func writeHTTPPoints(config *toml.Tree, con client.Client,
	url string, code int, bytes int, elapsed float64) {
	db := config.Get("influxdb.database").(string)
	measurement := config.Get("http.measurement").(string)
	srchost := config.Get("influxdb.srchost").(string)

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        db,
		Precision:       "s",
		RetentionPolicy: "autogen",
	})

	tags := map[string]string{
		"url":     url,
		"srchost": srchost,
	}
	fields := map[string]interface{}{
		"code":    code,
		"bytes":   bytes,
		"elapsed": elapsed,
	}

	pt, err := client.NewPoint(
		measurement,
		tags,
		fields,
		time.Now())

	bp.AddPoint(pt)

	err = con.Write(bp)
	perr(err)
}
