// main.go Command line entry point
// Copyright: 2018 Chris Luke
// License: MIT

package main

import (
	"flag"
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pelletier/go-toml"
	"log"
	"os"
	"time"
)

func main() {
	// Parse command line things
	config_file_p := flag.String("config", "config.toml", "Config file")
	verbose_p := flag.Bool("verbose", false, "Be verbose")
	debug_p := flag.Bool("debug", false, "Debugging output (implies verbose)")
	lookup_p := flag.String("lookup", "", "Lookup interface addresses "+
		"and exit (for diagnostic purposes)")

	flag.Parse()

	if *debug_p {
		*verbose_p = true
	}

	// Lookup the IPv4 and IPv6 addresses on the given interface
	// This is purely a diagnostic aid, to check that we detect
	// the correct IP addresses
	if len(*lookup_p) > 0 {
		ip, err := lookupAddrFromIface(*lookup_p, false)
		perr(err)
		if err == nil {
			fmt.Println("ipv4 addr", ip.String())
		}

		ip, err = lookupAddrFromIface(*lookup_p, true)
		perr(err)
		if err == nil {
			fmt.Println("ipv6 addr", ip.String())
		}
		os.Exit(0)
	}

	// Load the config file
	config, err := toml.LoadFile(*config_file_p)
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	// Soemwhat crude, but override values loaded from the config file
	// with those on the command line
	if *verbose_p {
		config.Set("core.verbose", *verbose_p)
	} else {
		value := config.Get("core.verbose")
		if value == nil {
			config.Set("core.verbose", false)
		}
	}

	if *debug_p {
		config.Set("core.debug", *debug_p)
	} else {
		value := config.Get("core.debug")
		if value == nil {
			config.Set("core.debug", false)
		}
	}

	// Open a connection to influxdb
	host := config.Get("influxdb.host").(string)
	port := config.Get("influxdb.port").(string)
	username := config.Get("influxdb.username").(string)
	password := config.Get("influxdb.password").(string)

	u := fmt.Sprintf("http://%s:%s", host, port)

	conf := client.HTTPConfig{
		Addr:     u,
		Username: username,
		Password: password,
	}

	con, err := client.NewHTTPClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Test the connection
	dur, ver, err := con.Ping(time.Second * 10)
	if err != nil {
		log.Fatal(err)
	}
	if *verbose_p {
		log.Printf("Connected to influxdb! %v, %s", dur, ver)
	}

	// Start the pinging threads
	go startIPv4Pinger(config, con)
	go startIPv6Pinger(config, con)
	readHTTPPoints(config, con)
}
