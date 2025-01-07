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

	Leader    State = "Leader"
	Candidate State = "Candidate"
	Follower  State = "Follower"
)

type State string

type Entry struct {
	Term    int
	Index   int
	Command interface{}
}

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
	term    int
	voteFor int
	votes   int

	logs []Entry

	commitIndex int
	lastApplied int

	nextIndex  []int
	matchIndex []int

	electionTimer  *time.Timer
	heartbeatTimer *time.Timer

	applyCh       chan<- ApplyMsg
	applyCond     *sync.Cond
	replicateCond []*sync.Cond
}

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

	rf.logs = make([]Entry, 1)
	rf.commitIndex = 0
	rf.lastApplied = 0

	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))

	rf.applyCh = applyCh
	rf.replicateCond = make([]*sync.Cond, len(rf.peers))

	// initialize from state persisted before a crash
	if rf.persister.RaftStateSize() > 0 {
		rf.readPersist(persister.ReadRaftState())
	}

	rf.electionTimer = time.NewTimer(randomElectionTimeout())
	rf.heartbeatTimer = time.NewTimer(heartbeatInterval)
	rf.applyCond = sync.NewCond(&rf.mu)

	for peer := range rf.peers {
		rf.nextIndex[peer], rf.matchIndex[peer] = rf.lastLog().Index+1, 0
		if peer != me {
			rf.replicateCond[peer] = sync.NewCond(&sync.Mutex{})
			go rf.replicater(peer)
		}
	}

	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.applier()

	return rf
}

func (rf *Raft) GetState() (int, bool) {
	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	return rf.term, rf.state == Leader
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
	// Your code here (3B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Leader {
		return -1, -1, false
	}
	newLogIndex := rf.lastLog().Index + 1
	rf.logs = append(rf.logs, Entry{
		Term:    rf.term,
		Index:   newLogIndex,
		Command: command,
	})

	rf.matchIndex[rf.me], rf.nextIndex[rf.me] = newLogIndex, newLogIndex+1
	for i := range rf.peers {
		if i != rf.me {
			rf.replicateCond[i].Signal()
		}
	}

	return newLogIndex, rf.term, true
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
		select {
		case <-rf.electionTimer.C:
			rf.mu.Lock()
			rf.election()
			rf.mu.Unlock()
		case <-rf.heartbeatTimer.C:
			rf.mu.Lock()
			if rf.state == Leader {
				rf.sendHeartbeat()
			}
			rf.mu.Unlock()
		}
	}
}

func (rf *Raft) replicater(peer int) {
	rf.replicateCond[peer].L.Lock()
	defer rf.replicateCond[peer].L.Unlock()

	for !rf.killed() {
		for !rf.needReplicate(peer) {
			rf.replicateCond[peer].Wait()
		}

		rf.replicate(peer)
	}
}

func (rf *Raft) applier() {
	for !rf.killed() {
		rf.mu.Lock()

		// if there are new committed entries, try apply
		for rf.commitIndex <= rf.lastApplied {
			rf.applyCond.Wait()
		}

		firstLogIndex, commitIndex, lastApplied := rf.firstLog().Index, rf.commitIndex, rf.lastApplied
		entries := make([]Entry, commitIndex-lastApplied)
		copy(entries, rf.logs[lastApplied-firstLogIndex+1:commitIndex-firstLogIndex+1])
		DPrintf("%v apply %v", rf.me, entries)
		rf.mu.Unlock()

		for _, entry := range entries {
			rf.applyCh <- ApplyMsg{
				CommandValid: true,
				Command:      entry.Command,
				CommandIndex: entry.Index,
			}
		}

		rf.mu.Lock()
		rf.lastApplied = commitIndex
		rf.mu.Unlock()

	}
}

func (rf *Raft) lastLog() Entry {
	return rf.logs[len(rf.logs)-1]
}

func (rf *Raft) firstLog() Entry {
	return rf.logs[0]
}
