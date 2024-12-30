package raft

type RequestVoteArgs struct {
	Candidate     int    // candidate ID
	CandidateTerm uint64 // candidate term
	LastLogIndex  uint64
	LastLogTerm   uint64
}

type RequestVoteReply struct {
	PeerTerm uint64 // follower term
	Granted  bool
}

func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Leader        int
	LeaderTerm    uint64
	CommitedIndex uint64
	PrevLogIndex  uint64
	PrevLogTerm   uint64
	Entries       []Entry
}

type AppendEntriesReply struct {
	FollowerTerm       uint64
	LastLogIndex       uint64
	ConflictTerm       uint64
	FirstConflictIndex uint64
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}
