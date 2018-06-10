// main.go Command line entry point
// Copyright: 2018 Chris Luke
// License: MIT

package main

import (
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/pelletier/go-toml"
	"log"
	"os"
	"time"
)

func main() {
	config, err := toml.LoadFile("config.toml")
	if err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}

	host := config.Get("influxdb.host").(string)
	port := config.Get("influxdb.port").(string)
	//measurement := config.Get("influxdb.measurement").(string)
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

	dur, ver, err := con.Ping(time.Second * 10)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connected to influxdb! %v, %s", dur, ver)

	go readIPv4PingPoints(config, con)
	go readIPv6PingPoints(config, con)
	readHTTPPoints(config, con)
}
