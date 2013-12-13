// Copyright 2013, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
vtworker is the main program to run a worker job.

It has two modes: single command or interactive.
- in single command, it will start the job passed in from the command line,
  and exit.
- in interactive mode, use a web browser to start an action.
*/
package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/youtube/vitess/go/vt/servenv"
	tm "github.com/youtube/vitess/go/vt/tabletmanager"
	"github.com/youtube/vitess/go/vt/topo"
	"github.com/youtube/vitess/go/vt/worker"
	"github.com/youtube/vitess/go/vt/wrangler"
)

var (
	port = flag.Int("port", 8080, "port for the status / interactive mode")
)

// signal handling, centralized here
func installSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		// we got a signal, notify our modules:
		// - tm will interrupt anything waiting on a tablet action
		// - wr will interrupt anything waiting on a shard or
		//   keyspace lock
		tm.SignalInterrupt()
		wrangler.SignalInterrupt()
		worker.SignalInterrupt()
	}()
}

var wrk worker.Worker

func main() {
	flag.Parse()
	args := flag.Args()

	installSignalHandlers()

	servenv.Init()
	defer servenv.Close()

	ts := topo.GetServer()
	defer topo.CloseServers()

	wr := wrangler.New(ts, 30*time.Second, 30*time.Second)
	if len(args) == 0 {
		// interactive mode, initialize the web UI to chose a command
		initInteractiveMode()
	} else {
		runCommand(wr, args)
	}

	servenv.Run(*port)
}