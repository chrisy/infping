// common stuff for infping and infhttp
// Copyright: 2018 Chris Luke
// License: MIT

package main

import (
	"fmt"
	"os"
)

func herr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func perr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
