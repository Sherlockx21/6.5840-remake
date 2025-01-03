package raft

type RequestVoteArgs struct {
	Candidate     int // candidate ID
	CandidateTerm int // candidate term
	LastLogIndex  int
	LastLogTerm   int
}

type RequestVoteReply struct {
	PeerTerm int // follower term
	Granted  bool
}

type AppendEntriesArgs struct {
	Leader        int
	LeaderTerm    int
	CommitedIndex int
	PrevLogIndex  int
	PrevLogTerm   int
	Entries       []Entry
}

type AppendEntriesReply struct {
	FollowerTerm       int
	LastLogIndex       int
	ConflictTerm       int
	FirstConflictIndex int
}
