package raft

type Log struct {
	entries []Entry // the first entry is the snapshot
}

type Entry struct {
	Term  int
	Index int
	Data  interface{}
}

func (rf *Raft) initLog() {
	l := &Log{
		entries: make([]Entry, 0),
	}
	l.entries = append(l.entries, Entry{Term: 0, Index: 0, Data: nil})
	rf.log = l
}

func (l *Log) lastIndex() int {
	return l.entries[len(l.entries)-1].Index
}

func (l *Log) term(logIndex int) int {
	arrayIndex := l.arrayIndex(logIndex)
	if arrayIndex < 0 {
		return -1
	}

	return l.entries[arrayIndex].Term
}

func (l *Log) arrayIndex(logIndex int) int {
	if logIndex > l.lastIndex() || logIndex < l.snapshot().Index {
		return -1
	}

	return logIndex - l.snapshot().Index
}

func (l *Log) snapshot() Entry {
	return l.entries[0]
}
