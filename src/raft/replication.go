package raft

import "sort"

func (rf *Raft) needReplicate(peer int) bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	// if leader's log is newer than target peer's, try replicate
	return rf.state == Leader && rf.matchIndex[peer] < rf.lastLog().Index
}

func (rf *Raft) replicate(peer int) {
	rf.mu.Lock()
	if rf.state != Leader {
		rf.mu.Unlock()
		return
	}
	args := rf.genAppendEntriesArgs(peer)
	rf.mu.Unlock()

	rf.sendAppendEntries(peer, args)
}

func (rf *Raft) genAppendEntriesArgs(peer int) *AppendEntriesArgs {
	prevLogIndex := rf.nextIndex[peer] - 1
	firstLogIndex := rf.firstLog().Index
	prevLogTerm := rf.logs[prevLogIndex-firstLogIndex].Term
	entries := make([]Entry, len(rf.logs[prevLogIndex-firstLogIndex+1:]))
	copy(entries, rf.logs[prevLogIndex-firstLogIndex+1:])
	args := &AppendEntriesArgs{
		Leader:       rf.me,
		LeaderTerm:   rf.term,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		LeaderCommit: rf.commitIndex,
		Entries:      entries,
	}
	return args
}

func (rf *Raft) sendAppendEntries(peer int, args *AppendEntriesArgs) {
	DPrintf("leader %v send entries %v to %v,", rf.me, args.Entries, peer)
	reply := &AppendEntriesReply{}
	if ok := rf.peers[peer].Call("Raft.AppendEntries", args, reply); !ok {
		DPrintf("leader %v send entries to %v connect failed", rf.me, peer)
		return
	}

	rf.handleAppendEntriesReply(peer, args, reply)
}

func (rf *Raft) handleAppendEntriesReply(peer int, args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.LeaderTerm != rf.term || rf.state != Leader {
		return
	}

	if !reply.Success {
		if reply.FollowerTerm > rf.term {
			rf.toState(Follower)
			rf.term, rf.voteFor = reply.FollowerTerm, None
			rf.persist()
		} else if reply.FollowerTerm == rf.term {
			// prepare for the next send, nextIndex to send should be the first conflicted index
			rf.nextIndex[peer] = reply.ConflictIndex
			if reply.ConflictTerm != -1 {
				// in this case, we need to find the index where leader's log start to conflict
				// with follower's, and try to overwrite them in the next replication
				firstLogIndex := rf.firstLog().Index
				for i := args.PrevLogIndex - 1; i >= firstLogIndex; i-- {
					if rf.logs[i-firstLogIndex].Term == reply.ConflictTerm {
						rf.nextIndex[peer] = i
						break
					}
				}
			}
		}
	} else {
		rf.matchIndex[peer] = args.PrevLogIndex + len(args.Entries)
		rf.nextIndex[peer] = rf.matchIndex[peer] + 1
		rf.updateLeaderCommit()
	}
}

func (rf *Raft) updateLeaderCommit() {
	sortedMatchIndex := make([]int, len(rf.matchIndex))
	copy(sortedMatchIndex, rf.matchIndex)
	sort.Ints(sortedMatchIndex)
	newCommitIndex := sortedMatchIndex[len(rf.matchIndex)/2]
	// newCommitIndex is commited by half of peers
	if newCommitIndex > rf.commitIndex {
		if rf.isLogMatched(newCommitIndex, rf.term) {
			DPrintf("%v update commitIndex to %v", rf.me, newCommitIndex)
			rf.commitIndex = newCommitIndex
			rf.applyCond.Signal()
		}
	}
}

func (rf *Raft) isLogMatched(index, term int) bool {
	// if we can find the log at index, and the term of this log is same as the term required,
	// then we regard the log is same
	return index <= rf.lastLog().Index && term == rf.logs[index-rf.firstLog().Index].Term
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.electionTimer.Reset(randomElectionTimeout()) // once peer get message from leader, update lastElectionTime to prevent electionTimeout

	reply.FollowerTerm = rf.term
	reply.Success = false

	if args.LeaderTerm < rf.term {
		return
	}

	if args.LeaderTerm > rf.term {
		rf.toState(Follower)
		rf.term, rf.voteFor = args.LeaderTerm, None
		rf.persist()
	}

	if args.PrevLogIndex < rf.firstLog().Index { // log at index not found in follower
		return
	}

	if !rf.isLogMatched(args.PrevLogIndex, args.PrevLogTerm) {
		lastLogIndex := rf.lastLog().Index
		if lastLogIndex < args.PrevLogIndex { // all logs from leader are new to follower
			reply.ConflictIndex, reply.ConflictTerm = lastLogIndex+1, -1
		} else {
			firstLogIndex := rf.firstLog().Index
			index := args.PrevLogIndex
			for index >= firstLogIndex && rf.logs[index-firstLogIndex].Term == args.PrevLogTerm {
				index--
			}
			reply.ConflictIndex, reply.ConflictTerm = index+1, args.PrevLogTerm
		}
		return
	}

	// allow to append entries
	firstLogIndex := rf.firstLog().Index
	for i, entry := range args.Entries {
		if entry.Index-firstLogIndex >= len(rf.logs) || rf.logs[entry.Index-firstLogIndex].Term != entry.Term {
			rf.logs = append(rf.logs[:entry.Index-firstLogIndex], args.Entries[i:]...)
			rf.persist()
			break
		}
	}

	newCommitIndex := args.LeaderCommit
	if rf.lastLog().Index < args.LeaderCommit {
		newCommitIndex = rf.lastLog().Index
	}
	if newCommitIndex > rf.commitIndex {
		rf.commitIndex = newCommitIndex
		rf.applyCond.Signal()
	}
	reply.Success = true
}
