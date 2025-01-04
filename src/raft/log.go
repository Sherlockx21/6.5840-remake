package raft

type Logs struct {
	entries []Entry // the first entry is the snapshot
}

type Entry struct {
	Term    int
	Index   int
	Command interface{}
}

func (rf *Raft) initLogs() {
	rf.logs.entries = make([]Entry, 1)
}

func (l *Logs) lastLog() Entry {
	return l.entries[len(l.entries)-1]
}
