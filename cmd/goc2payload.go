package goc2payload

import (
	"flag"

	"github.com/grines/goc2-agent/pkg/linux"
)

var (
	typePtr string
)

//Start RedMap
func Start() {
	//flags
	flag.StringVar(&typePtr, "type", "linux", "Build Type")
	flag.Parse()

	if typePtr == "linux" {
		linux.Build()
	}

}
