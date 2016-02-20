package main

import "fmt"
import "testing"

var sm *StateMachine
func Test_AppendEntriesReq(t *testing.T) {
	// testing testing
	var exp interface{}
	data := []byte{1,0,0,1,1,0,0,0,1,1}
	entry := []logEntries{{term: 1, data: data}, {term: 2, data: data}}
	lg := []logEntries{{},{term: 1, data: data}}
	sm = &StateMachine{id: 3, term: 2, votedFor: 0, log: lg, commitIndex: 0, state: "follower"}
	actions := sm.ProcessEvent(AppendEntriesReqEv{term : 1, leaderId: 1, prevLogIndex: 1, prevLogTerm: 1, entries: entry, leaderCommit: 1, from: 1})
	exp = Send{1, AppendEntriesRespEv{2,false,3}}
	expect(t, actions[0], exp)
	//fmt.Println(actions)
	actions = sm.ProcessEvent(AppendEntriesReqEv{term : 3, leaderId: 1, prevLogIndex: 1, prevLogTerm: 1, entries: entry, leaderCommit: 1, from: 1})
	exp = LogStoreEv{2, entry[0], 3}
	//fmt.Println(exp)
	//expect(t, actions[0], exp)
	exp = LogStoreEv{3, entry[1], 3}
	//expect(t, actions[1], exp)
	exp = Send{1, AppendEntriesRespEv{3,true,3}}
	expect(t, actions[2], exp)
	//fmt.Println(actions)
	sm = &StateMachine{id: 3, term: 2, votedFor: 0, log: lg, commitIndex: 0, state: "candidate"}
	actions = sm.ProcessEvent(AppendEntriesReqEv{term : 3, leaderId: 1, prevLogIndex: 1, prevLogTerm: 1, entries: entry, leaderCommit: 1, from: 1})
	exp = Send{1, AppendEntriesRespEv{3,true,3}}
	expect(t, actions[0], exp)
	s_expect(t, sm.state, "follower")
	
}
func Test_AppendEntriesResp(t *testing.T){
	nI := [6]int{0,5,3,4,3,2}
	mI := [6]int{0,4,2,3,2,1}
	sm = &StateMachine{id: 1, term: 3, state: "leader", lastLogIndex: 4, nextIndex: nI, matchIndex: mI}
	actions := sm.ProcessEvent(AppendEntriesRespEv{term: 2, success: true, from: 4})
	s_expect(t, string(sm.nextIndex[4]), string(sm.lastLogIndex+1))
	s_expect(t, string(sm.matchIndex[4]), string(sm.lastLogIndex))
	//fmt.Println(sm)
	//actions = sm.ProcessEvent(AppendEntriesRespEv{term: 2, success: false, from: 3})
	//exp = Send{4, AppendEntriesReqEv{3, 1, 1, }}
	//fmt.Println(actions)
	//fmt.Println(sm)
	actions = sm.ProcessEvent(AppendEntriesRespEv{term: 4, success: false, from: 4})
	s_expect(t, sm.state, "follower")
	//fmt.Println(actions)
	

}

func Test_VoteReq(t *testing.T){
	sm = &StateMachine{id: 2, term: 2, votedFor: 0, lastLogTerm: 2, lastLogIndex: 7, state: "follower"}
	actions := sm.ProcessEvent(VoteReqEv{term: 3, candidateId: 4, lastLogIndex: 9, lastLogTerm: 3, from: 4})
	exp := Send{4, VoteRespEv{3, true, 2}}
	expect(t, actions[0], exp)
	sm = &StateMachine{id: 2, term: 2, votedFor: 1, lastLogTerm: 2, lastLogIndex: 7, state: "follower"}
	actions = sm.ProcessEvent(VoteReqEv{term: 3, candidateId: 4, lastLogIndex: 9, lastLogTerm: 3, from: 4})
	exp = Send{4, VoteRespEv{2, false, 2}}
	expect(t, actions[0], exp)
	sm = &StateMachine{id: 2, term: 2, votedFor: 1, lastLogTerm: 2, lastLogIndex: 7, state: "follower"}
	actions = sm.ProcessEvent(VoteReqEv{term: 3, candidateId: 4, lastLogIndex: 9, lastLogTerm: 1, from: 4})
	exp = Send{4, VoteRespEv{2, false, 2}}
	expect(t, actions[0], exp)
	sm = &StateMachine{id: 2, term: 2, votedFor: 1, lastLogTerm: 2, lastLogIndex: 7, state: "leader"} // or candidate
	actions = sm.ProcessEvent(VoteReqEv{term: 3, candidateId: 4, lastLogIndex: 9, lastLogTerm: 1, from: 4})
	s_expect(t, sm.state, "follower")
	//fmt.Println(actions)
	// if already voted
	// for diff values of lastindex and last term
	// for leader and candidate
	//fmt.Println(sm)
}
func Test_VoteResp(t *testing.T){
	r := [6]int{0,0,1,0,0,-1}
	var exp interface{}
	sm = &StateMachine{id: 2, peers: [4]int{1,3,4,5}, term: 4, votedFor: 2, numVotesGranted: 1, numVotesDenied: 1, votesRcvd: r, state: "candidate"}
	actions := sm.ProcessEvent(VoteRespEv{term: 3, voteGranted: true, from: 4})
	actions = sm.ProcessEvent(VoteRespEv{term: 3, voteGranted: true, from: 3})
	s_expect(t, sm.state, "leader")
	for _,v := range sm.peers{
		exp = Send{v, AppendEntriesReqEv{term: 4,leaderId: 2, from: 2}}
		//expect(t, actions[i], exp)
	}
	exp = alarm{200}
	expect(t, actions[4], exp)
	
	//fmt.Println(actions)
	sm = &StateMachine{id: 5, term: 4, votedFor: 2, numVotesGranted: 1, numVotesDenied: 1, votesRcvd: r, state: "candidate"}
	actions = sm.ProcessEvent(VoteRespEv{term: 5, voteGranted: true, from: 3})
	s_expect(t, sm.state, "follower")
	// for 3 values of votegranted
	// for chhota term
	// for majority and win
	//fmt.Println(sm)
}

func expect(t *testing.T, a interface{}, b interface{}){
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}
func s_expect(t *testing.T, a string, b string){
	if a != b {
		t.Error(fmt.Sprintf("Expected %v, found %v", b, a)) // t.Error is visible when running `go test -verbose`
	}
}