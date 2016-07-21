package main

import (
	"os/signal"
	"syscall"
)

func ignoreSigPipe() {
	signal.Ignore(syscall.SIGPIPE)
}
