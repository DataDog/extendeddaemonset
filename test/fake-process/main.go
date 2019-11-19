// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	readinessDelay time.Duration
	startTime      time.Time
)

func init() {
	startTime = time.Now()

	readinessDelay = 30 * time.Second
	delayString := os.Getenv("READINESS_DELAY")
	if delayString != "" {
		var err error
		readinessDelay, err = time.ParseDuration(delayString)
		if err != nil {
			log.Fatalf("unable to parse readiness delay, err:%v", err)
			os.Exit(1)
		}
	}
}

func main() {
	http.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		if now.Sub(startTime.Add(readinessDelay)) < 0 {
			http.Error(w, "process not ready", http.StatusPreconditionFailed)
		} else {
			fmt.Fprintf(w, "Process Ready return at, %v", now)
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
