package raft

func (rf *Raft) genAppendEntriesArgs(peer int) *AppendEntriesArgs {
	args := &AppendEntriesArgs{
		Leader:     rf.me,
		LeaderTerm: rf.term,
	}
	return args
}

func (rf *Raft) sendAppendEntries(peer int, args *AppendEntriesArgs) {
	DPrintf("leader %v send append entries to %v, and is a heartbeat? %v", rf.me, peer, args.Entries == nil)
	reply := &AppendEntriesReply{}
	if ok := rf.peers[peer].Call("Raft.AppendEntries", args, reply); !ok {
		DPrintf("leader %v send entries to %v connect failed", rf.me, peer)
		return
	}

	rf.handleAppendEntriesReply(reply)
}

func (rf *Raft) handleAppendEntriesReply(reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if reply.FollowerTerm > rf.term {
		rf.toState(Follower)
		rf.term, rf.voteFor = reply.FollowerTerm, None
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.electionTimer.Reset(randomElectionTimeout()) // once peer get message from leader, update lastElectionTime to prevent electionTimeout

	reply.FollowerTerm = rf.term
	reply.LastLogIndex = rf.logs.lastLog().Index

	if args.LeaderTerm < rf.term {
		return
	}

	if args.LeaderTerm > rf.term {
		rf.toState(Follower)
		rf.term, rf.voteFor = args.LeaderTerm, None
	}
}
