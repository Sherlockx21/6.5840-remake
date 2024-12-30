package raft

import (
	"time"
)

func (rf *Raft) shouldStartElection() bool {
	return time.Since(rf.lastElection) > rf.electionTimeout
}

func (rf *Raft) election() {
	DPrintf("peer %v start election", rf.me)
	rf.toCandidate()
	args := &RequestVoteArgs{
		Candidate:     rf.me,
		CandidateTerm: rf.term,
	}

	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go rf.askForVote(i, args)
	}
}

func (rf *Raft) toCandidate() {
	defer rf.persist()
	rf.state = Follower
	rf.term++
	rf.voteFor = rf.me
	rf.votes = 1
	rf.resetElectionTimer()
}

func (rf *Raft) toLeader() {
	DPrintf("peer %v become leader", rf.me)
	rf.state = Leader
	// reset tracker
}

func (rf *Raft) toFollower(term uint64) {
	rf.state = Follower
	if term > rf.term {
		rf.term = term
		rf.voteFor = None
		rf.votes = 0
		rf.persist()
	}
}

func (rf *Raft) askForVote(peer int, args *RequestVoteArgs) {
	reply := &RequestVoteReply{}
	if ok := rf.sendRequestVote(peer, args, reply); !ok {
		DPrintf("candidate %v ask %v connect failed", rf.me, peer)
		return
	}

	rf.handleVoteReply(reply)
}

func (rf *Raft) handleVoteReply(reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if reply.PeerTerm > rf.term {
		rf.toFollower(reply.PeerTerm)
		return
	}

	if reply.Granted {
		rf.votes++
		if rf.votes > len(rf.peers)/2 {
			rf.toLeader()
		}
	}
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Granted = false
	reply.PeerTerm = rf.term

	if args.CandidateTerm < rf.term {
		return
	}

	if args.CandidateTerm > rf.term {
		rf.toFollower(args.CandidateTerm)
		reply.PeerTerm = rf.term
	}

	if (rf.voteFor == None || rf.voteFor == args.Candidate) && true {
		rf.voteFor = args.Candidate
		reply.Granted = true
		rf.resetElectionTimer()
		return
	}
}

func (rf *Raft) selfAlive() bool {
	return true
}

func (rf *Raft) shouldDoHeartbeat() bool {
	return time.Since(rf.lastHeartbeat) > heartbeatInterval
}

func (rf *Raft) sendHeartbeat() {
	args := &AppendEntriesArgs{
		Leader:     rf.me,
		LeaderTerm: rf.term,
	}
	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		go rf.doAppendEntries(i, args)
	}
}
