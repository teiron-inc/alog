package main

import (
	"github.com/teiron-inc/alog"
)

func main() {
	alog.RegisterAlog()
	alog.SetLogTag("CONSOLE")
	for i := 0; i < 10; i++ {
		alog.InfoC("The console:", i)
	}
}
