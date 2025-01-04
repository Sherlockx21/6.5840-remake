package raft

func (rf *Raft) election() {
	rf.toState(Candidate)
	DPrintf("peer %v start election with term %v ", rf.me, rf.term)
	args := rf.genRequestVoteArgs()

	// now ask for vote in parallel
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		go rf.askForVote(i, args)
	}
}

func (rf *Raft) toState(state State) {
	rf.state = state
	switch state {
	case Follower:
		rf.heartbeatTimer.Stop()
		rf.electionTimer.Reset(randomElectionTimeout())
	case Candidate:
		rf.term++
		rf.voteFor = rf.me
		rf.votes = 1
		rf.electionTimer.Reset(randomElectionTimeout())
	case Leader:
		rf.electionTimer.Stop()
		rf.heartbeatTimer.Reset(heartbeatInterval)
	}
}

func (rf *Raft) genRequestVoteArgs() *RequestVoteArgs {
	return &RequestVoteArgs{
		Candidate:     rf.me,
		CandidateTerm: rf.term,
		LastLogIndex:  rf.logs.lastLog().Index,
		LastLogTerm:   rf.logs.lastLog().Term,
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
		rf.toState(Follower)
		rf.term, rf.voteFor = reply.PeerTerm, None
		return
	}

	if reply.Granted {
		rf.votes++
		DPrintf("%v get %v votes in term %v", rf.me, rf.votes, rf.term)
		if rf.votes > len(rf.peers)/2 {
			rf.toState(Leader)
			rf.sendHeartbeat()
		}
	}
}

func (rf *Raft) sendHeartbeat() {
	rf.heartbeatTimer.Reset(heartbeatInterval)
	for i := range rf.peers {
		if i == rf.me {
			continue
		}

		args := rf.genAppendEntriesArgs(i)
		go rf.sendAppendEntries(i, args)
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
		rf.toState(Follower)
		rf.term, rf.voteFor = args.CandidateTerm, None
		reply.PeerTerm = rf.term
	}

	// If votedFor is null or candidateId, and candidate’s log is at
	// least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
	if (rf.voteFor == None || rf.voteFor == args.Candidate) && rf.isLogUptoDate(args) {
		rf.voteFor = args.Candidate
		reply.Granted = true
		return
	}
}

func (rf *Raft) isLogUptoDate(args *RequestVoteArgs) bool {
	if args.LastLogTerm > rf.logs.lastLog().Term ||
		(args.LastLogTerm == rf.logs.lastLog().Term && args.LastLogIndex >= rf.logs.lastLog().Index) {
		return true
	}
	return false
}
