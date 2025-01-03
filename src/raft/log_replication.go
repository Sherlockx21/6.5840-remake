package raft

import "time"

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.lastElection = time.Now() // once peer get message from leader, update lastElectionTime to prevent electionTimeout

	reply.FollowerTerm = rf.term
	reply.LastLogIndex = rf.log.lastIndex()

	if args.LeaderTerm < rf.term {
		return
	}

	if args.LeaderTerm > rf.term {
		rf.toFollower(args.LeaderTerm)
		reply.FollowerTerm = rf.term
	}
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

	tmp := reply
	tmp.ConflictTerm = 0
}
