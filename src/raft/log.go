package raft

type Entry struct {
	Term  uint64
	Index uint64
	Data  interface{}
}
