package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

const (
	tickInterval        = 50 * time.Millisecond
	heartbeatInterval   = 100 * time.Millisecond
	baseElectionTimeout = 200

	None = -1

	Leader State = iota
	Candidate
	Follower
)

type State int

type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	state   State
	term    uint64
	voteFor int
	votes   int

	electionTimeout time.Duration
	lastElection    time.Time
	lastHeartbeat   time.Time

	// log
	// tracker

	applyCh chan<- ApplyMsg
}

// Use for test
func (rf *Raft) GetState() (int, bool) {
	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return int(rf.term), !rf.killed() && rf.state == Leader
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
}

func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {
	for !rf.killed() {
		rf.mu.Lock()

		switch rf.state {
		case Leader:
			if !rf.selfAlive() {
				rf.toFollower(rf.term)
				break
			}

			if rf.shouldDoHeartbeat() {
				rf.lastHeartbeat = time.Now()
				rf.sendHeartbeat()
			}

		case Candidate, Follower:
			if rf.shouldStartElection() {
				rf.election()
			}
		}

		rf.mu.Unlock()
		time.Sleep(tickInterval)
	}
}

// Create peer instance
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (3A, 3B, 3C).
	rf.state = Follower
	rf.term = 0
	rf.voteFor = None
	rf.votes = 0

	// rf.log = newlog()
	// rf.tracker = newtracker()
	rf.applyCh = applyCh

	// initialize from state persisted before a crash
	if rf.persister.RaftStateSize() > 0 {
		rf.readPersist(persister.ReadRaftState())
	}

	// start ticker goroutine to start elections
	go rf.ticker()
	// go rf.commiter()

	return rf
}

func (rf *Raft) resetElectionTimer() {
	ms := baseElectionTimeout + rand.Int63()%baseElectionTimeout
	rf.electionTimeout = time.Duration(ms) * time.Millisecond
	rf.lastElection = time.Now()
}
