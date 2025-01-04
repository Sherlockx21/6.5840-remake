package raft

import (
	"log"
	"math/rand"
	"time"
)

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

func randomElectionTimeout() time.Duration {
	return time.Duration(baseElectionTimeout+rand.Int63()%baseElectionTimeout) * time.Millisecond
}
