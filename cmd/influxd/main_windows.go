package main

func ignoreSigPipe() {
	// ignoreSigPipe does nothing on Windows since there is no SIGPIPE.
}
