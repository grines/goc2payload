package portscan

import "github.com/JustinTimperio/gomap"

func Portscan() string {
	print("Scanning")
	var (
		proto    = "tcp"
		fastscan = true
		syn      = false
	)

	scan, err := gomap.ScanRange(proto, fastscan, syn)
	if err != nil {
		// handle error
	}
	return scan.String()
}
