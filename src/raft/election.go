package raft

import (
	"time"
)

// if candidate or follower's election timeout and not recieve from leader, start election
func (rf *Raft) shouldStartElection() bool {
	// lastElection would be updated once peer receives rpc message from leader
	return time.Since(rf.lastElection) > rf.electionTimeout
}

func (rf *Raft) election() {
	rf.toCandidate()
	DPrintf("peer %v start election with term %v ", rf.me, rf.term)
	args := &RequestVoteArgs{
		Candidate:     rf.me,
		CandidateTerm: rf.term,
		LastLogIndex:  rf.log.lastIndex(),
		LastLogTerm:   rf.log.term(rf.log.lastIndex()),
	}

	// now ask for vote in parallel
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go rf.askForVote(i, args)
	}
}

// As described in paper, to begin an election, a follower increments its current
// term and transitions to candidate state. It then votes for itself and issues
// RequestVote RPCs in parallel to each of the other servers in the cluster.
func (rf *Raft) toCandidate() {
	rf.state = Candidate
	rf.term++
	rf.voteFor = rf.me
	rf.votes = 1
	rf.resetElectionTimer()
	rf.persist()
}

func (rf *Raft) toLeader() {
	DPrintf("peer %v become leader", rf.me)
	rf.state = Leader
}

func (rf *Raft) toFollower(term int) {
	DPrintf("%v change to follower", rf.me)
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
	if ok := rf.peers[peer].Call("Raft.RequestVote", args, reply); !ok {
		DPrintf("cand %v ask peer %v for vote connect failed", rf.me, peer)
		return
	}

	rf.handleVoteReply(reply)
}

func (rf *Raft) handleVoteReply(reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != Candidate { // peer's state might change if it recive heartbeat from a newer leader during election
		return
	}

	if reply.PeerTerm > rf.term {
		rf.toFollower(reply.PeerTerm)
		return
	}

	if reply.Granted {
		rf.votes++
		DPrintf("%v get %v votes in term %v", rf.me, rf.votes, rf.term)
		if rf.votes > len(rf.peers)/2 {
			rf.toLeader()
			rf.sendHeartbeat()
		}
	}
}

func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Granted = false
	reply.PeerTerm = rf.term

	if args.CandidateTerm < rf.term { // Reply false if term < currentTerm (§5.1)
		return
	}

	if args.CandidateTerm > rf.term {
		rf.toFollower(args.CandidateTerm)
		reply.PeerTerm = rf.term
	}

	// If votedFor is null or candidateId, and candidate’s log is at
	// least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
	if (rf.voteFor == None || rf.voteFor == args.Candidate) && rf.isLogUptoDate(args) {
		DPrintf("%v vote for %v in term %v", rf.me, args.Candidate, rf.term)
		rf.voteFor = args.Candidate
		//rf.lastElection = time.Now()
		reply.Granted = true
		return
	}
}

func (rf *Raft) isLogUptoDate(args *RequestVoteArgs) bool {
	if args.LastLogTerm > rf.log.term(rf.log.lastIndex()) ||
		(args.LastLogTerm == rf.log.term(rf.log.lastIndex()) && args.LastLogIndex >= rf.log.lastIndex()) {
		return true
	}
	return false
}

func (rf *Raft) shouldDoHeartbeat() bool {
	return time.Since(rf.lastHeartbeat) > heartbeatInterval
}

func (rf *Raft) sendHeartbeat() {
	args := &AppendEntriesArgs{
		Leader:     rf.me,
		LeaderTerm: rf.term,
		Entries:    nil,
	}
	rf.lastHeartbeat = time.Now()

	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		go rf.sendAppendEntries(i, args)
	}
}
