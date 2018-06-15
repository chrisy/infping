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
	"net/http"
	"time"
)

func readHTTPPoints(config *toml.Tree, con client.Client) {
	verbose := config.Get("core.verbose").(bool)
	urls := config.Get("http.urls").([]interface{})
	if verbose {
		log.Printf("Going to fetch the following urls: %q", urls)
	}
	for {
		for _, v := range urls {
			url, _ := v.(string)
			go func(url string) {
				start := time.Now()
				response, err := http.Get(url)
				perr(err)
				contents, err := ioutil.ReadAll(response.Body)
				defer response.Body.Close()
				perr(err)
				elapsed := time.Since(start).Seconds()
				code := response.StatusCode
				bytes := len(contents)
				if verbose {
					log.Printf("Url:%s, code: %s, bytes: %s, elapsed: %s", url, code, bytes, elapsed)
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
