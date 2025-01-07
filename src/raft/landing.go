package raft

import (
	"bytes"

	"6.5840/labgob"
)

func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.term)
	e.Encode(rf.voteFor)
	e.Encode(rf.logs)
	raftstate := w.Bytes()
	rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var term, voteFor int
	var entries []Entry
	if d.Decode(&term) != nil || d.Decode(&voteFor) != nil || d.Decode(&entries) != nil {
		DPrintf("%v decode states and snapshot failed", rf.me)
	} else {
		rf.term = term
		rf.voteFor = voteFor
		rf.logs = entries
		rf.lastApplied = rf.firstLog().Index
		rf.commitIndex = rf.firstLog().Index
	}
}
