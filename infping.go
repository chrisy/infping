// infping.go Ping probes via 'fping'
// Copyright: 2016 Tor Hveem
// copyright: 2018 Chris Luke
// License: MIT

package main

import (
	"bufio"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pelletier/go-toml"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func slashSplitter(c rune) bool {
	return c == '/'
}

func startIPv4Pinger(config *toml.Tree, con client.Client) {
	for {
		readPingPoints(config, con, "ipv4", "fping")
		time.Sleep(time.Second * 1)
	}
}

func startIPv6Pinger(config *toml.Tree, con client.Client) {
	for {
		readPingPoints(config, con, "ipv6", "fping6")
		time.Sleep(time.Second * 1)
	}
}

func readPingPoints(config *toml.Tree, con client.Client, family string, fping string) {
	verbose := config.Get("core.verbose").(bool)
	hosts := config.Get("ping." + family + "_hosts").([]interface{})
	srcaddr := config.Get("ping." + family + "_srcaddr").(string)

	if verbose {
		log.Printf("Going to ping the following hosts: %q", hosts)
	}

	args := []string{"-B 1", "-D", "-r0", "-O 0", "-Q 10", "-p 1000", "-l"}

	if len(srcaddr) > 0 {
		// Add a source address
		args = append(args, "-S "+srcaddr)
	}

	for _, v := range hosts {
		host, _ := v.(string)
		args = append(args, host)
	}
	cmd := exec.Command("/usr/bin/"+fping, args...)
	stdout, err := cmd.StdoutPipe()
	herr(err)
	stderr, err := cmd.StderrPipe()
	herr(err)
	cmd.Start()
	perr(err)

	buff := bufio.NewScanner(stderr)
	for buff.Scan() {
		text := buff.Text()
		fields := strings.Fields(text)
		// Ignore timestamp
		if len(fields) > 1 {
			host := fields[0]
			data := fields[4]
			dataSplitted := strings.FieldsFunc(data, slashSplitter)

			// Remove ,
			dataSplitted[2] = strings.TrimRight(dataSplitted[2], "%,")
			sent, recv, lossp := dataSplitted[0], dataSplitted[1], dataSplitted[2]
			min, max, avg := "", "", ""

			// Ping times
			if len(fields) > 5 {
				times := fields[7]
				td := strings.FieldsFunc(times, slashSplitter)
				min, avg, max = td[0], td[1], td[2]
			}

			if verbose {
				log.Printf("%s Host:%s, loss: %s, min: %s, avg: %s, max: %s",
					family, host, lossp, min, avg, max)
			}
			writePingPoints(config, con, host, family, sent, recv, lossp, min, avg, max)
		}
	}

	std := bufio.NewReader(stdout)
	line, err := std.ReadString('\n')
	perr(err)

	if verbose {
		log.Printf("%s died; stdout: %s", fping, line)
	}
}

func writePingPoints(config *toml.Tree, con client.Client,
	host string, af string, sent string, recv string,
	lossp string, min string, avg string, max string) {
	db := config.Get("influxdb.database").(string)
	measurement := config.Get("ping.measurement").(string)
	srchost := config.Get("influxdb.srchost").(string)

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:        db,
		Precision:       "s",
		RetentionPolicy: "autogen",
	})

	tags := map[string]string{
		"host":    host,
		"srchost": srchost,
		"af":      af,
	}

	fields := map[string]interface{}{}
	loss, _ := strconv.Atoi(lossp)
	if min != "" && avg != "" && max != "" {
		min, _ := strconv.ParseFloat(min, 64)
		avg, _ := strconv.ParseFloat(avg, 64)
		max, _ := strconv.ParseFloat(max, 64)
		fields = map[string]interface{}{
			"loss": loss,
			"min":  min,
			"avg":  avg,
			"max":  max,
		}
	} else {
		fields = map[string]interface{}{
			"loss": loss,
		}
	}

	pt, err := client.NewPoint(
		measurement,
		tags,
		fields,
		time.Now())

	bp.AddPoint(pt)

	err = con.Write(bp)
	if err != nil {
		log.Fatal(err)
	}
}
