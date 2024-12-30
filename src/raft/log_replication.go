package raft

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.resetElectionTimer()
}

func (rf *Raft) doAppendEntries(peer int, args *AppendEntriesArgs) {
	reply := &AppendEntriesReply{}
	if ok := rf.sendAppendEntries(peer, args, reply); !ok {
		DPrintf("leader %v send entries to %v connect failed", rf.me, peer)
		return
	}

	rf.handleAppendEntriesReply(reply)
}

func (rf *Raft) handleAppendEntriesReply(reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	tmp := reply
	tmp.ConflictTerm = 0
}
