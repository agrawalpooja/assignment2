package main
//import "fmt"
import "math"
type logEntries struct{
		term int
		data []byte
	}
type VoteReqEv struct {
	candidateId int
	term int
	lastLogIndex int
	lastLogTerm int
	from int
	// etc
}
type VoteRespEv struct {
	term int
	voteGranted bool
	from int
	// etc
}
type TimeoutEv struct {
}
type alarm struct{
	sec int
}
type AppendEv struct{
	index int
	data []byte
}
type CommitEv struct{

}
type Send struct{
	peerId int
	event interface{}
}
type AppendEntriesReqEv struct {
	term int
	leaderId int
	prevLogIndex int
	prevLogTerm int
	entries []logEntries
	leaderCommit int
	from int
	// etc
}
type AppendEntriesRespEv struct {
	term int
	success bool
	from int
	// etc
}
type LogStoreEv struct{
	index int
	entry logEntries
	from int
}

type StateMachine struct {
	id int // server id
	peers [4]int // other server ids
	term int 
	votedFor int
	log []logEntries
	commitIndex int
	lastApplied int
	lastLogTerm int
	lastLogIndex int
	nextIndex [6]int
	matchIndex [6]int
	majority int
	state string
	numVotesGranted, numVotesDenied int
	votesRcvd [6]int // one per peer
	// etc
}



func (sm *StateMachine) ProcessEvent (ev interface{}) []interface{}{
	var actions []interface{}
	switch ev.(type) {

		// handle heartbeat ?
	case AppendEntriesReqEv:
		cmd := ev.(AppendEntriesReqEv)
		if sm.state == "follower"{
			if cmd.term < sm.term || sm.log[cmd.prevLogIndex].term != cmd.prevLogTerm{
				actions = append(actions, Send{cmd.from, AppendEntriesRespEv{term: sm.term, success: false, from: sm.id}})
			}else{
				i := cmd.prevLogIndex
				for _,v := range cmd.entries{
					i = i + 1
					//sm.log[i] = v //logstore
					actions = append(actions, LogStoreEv{index: i, entry: v, from: sm.id})
				}
				if cmd.leaderCommit > sm.commitIndex{
					sm.commitIndex = int(math.Min(float64(cmd.leaderCommit), float64(i)))
				}
				if cmd.term > sm.term{
					sm.term = cmd.term
					sm.votedFor = 0
				}
				actions = append(actions, Send{cmd.from, AppendEntriesRespEv{term: sm.term, success: true, from: sm.id}})
			}
		}
		//check if term changes ?
		if sm.state == "candidate"{
			sm.state = "follower"
			sm.term = cmd.term
			sm.votedFor = 0
			actions = append(actions, Send{cmd.from, AppendEntriesRespEv{term: sm.term, success: true, from: sm.id}})
		}
		// do stuff with req
		//fmt.Printf("%v\n", cmd)

	case AppendEntriesRespEv:
		cmd := ev.(AppendEntriesRespEv)
		if sm.state == "leader"{
			if cmd.success == true{
				//update nextindex and matchindex // how? from id?
				sm.nextIndex[cmd.from] = sm.lastLogIndex + 1
				sm.matchIndex[cmd.from] = sm.lastLogIndex
			}else{
				// two conditions
				if sm.term < cmd.term{
					sm.state = "follower"
					sm.term = cmd.term
					sm.votedFor = 0
				}else{
					sm.nextIndex[cmd.from]--
					prev := sm.nextIndex[cmd.from] - 1
					actions = append(actions, Send{cmd.from ,AppendEntriesReqEv{term: sm.term, leaderId: sm.id, prevLogIndex: prev, prevLogTerm: sm.log[prev].term, entries: sm.log[prev:], leaderCommit: sm.commitIndex, from: sm.id}})
				}
				//decrement nextIndex and retry ////////prevlogindex?
			}
		}
	
	// RequestVote RPC
	case VoteReqEv:
		cmd := ev.(VoteReqEv)
		var voteGranted bool
		if sm.state == "follower"{
			if cmd.term < sm.term{
				voteGranted = false
			}else if sm.votedFor == 0 || sm.votedFor == cmd.candidateId{
					if cmd.lastLogTerm > sm.lastLogTerm{		//sm.log[len(sm.log)].term
						sm.term = cmd.term
						sm.votedFor = cmd.candidateId
						voteGranted = true
					}else if cmd.lastLogTerm == sm.lastLogTerm && cmd.lastLogIndex >= sm.lastLogIndex{		//len(sm.log)
						sm.term = cmd.term
						sm.votedFor = cmd.candidateId
						voteGranted = true
					}else{
						voteGranted = false
					}
			}else{
				voteGranted =false
			}
			actions = append(actions, Send{cmd.from, VoteRespEv{term: sm.term, voteGranted: voteGranted, from: sm.id}})
			// do stuff with req
			//fmt.Printf("%v\n", cmd)
		}else{										//leader or candidate
				if sm.term < cmd.term{
					sm.term = cmd.term
					sm.votedFor = 0
					sm.state = "follower"     //do we need resp here?
				}
		}
	case VoteRespEv:
		cmd := ev.(VoteRespEv)
		if sm.state == "candidate"{
			if sm.term < cmd.term{
				sm.term = cmd.term
				sm.votedFor = 0
				sm.state = "follower"
			}else{
				if cmd.voteGranted == true{
					sm.numVotesGranted++
					sm.votesRcvd[cmd.from] = 1		// 1 for peer has voted yes
				}else if cmd.voteGranted == false{
					sm.numVotesDenied++		
					sm.votesRcvd[cmd.from] = -1		// -1 for peer has voted no
				}else{
					sm.votesRcvd[cmd.from] = 0		// 0 for peer has not voted
				}
				sm.majority = 3
				if sm.numVotesGranted >= sm.majority{
					sm.state = "leader"
					for _,i := range sm.peers{
						actions = append(actions, Send{i ,AppendEntriesReqEv{term: sm.term, leaderId: sm.id, leaderCommit: sm.commitIndex, from: sm.id}}) //to all
					}
					actions = append(actions, alarm{sec: 200})  //heartbeat timer
				}
			}
		}	

	case TimeoutEv:
		//cmd := ev.(TimeoutEv)
		//election timeout
		if sm.state == "follower" || sm.state == "candidate"{
			sm.state = "candidate"
			sm.term++
			sm.votedFor = sm.id
			//reset election timer
			actions = append(actions, alarm{sec: 500})
			//n := len(sm.log)
			for _,i := range sm.peers{
				actions = append(actions, Send{i, VoteReqEv{term: sm.term, candidateId: sm.id, lastLogIndex: sm.lastLogIndex, lastLogTerm: sm.lastLogTerm, from: sm.id}})//to all			
			}
			
		}else{
			for _,i := range sm.peers{
				actions = append(actions, Send{i, AppendEntriesReqEv{term: sm.term, leaderId: sm.id, leaderCommit: sm.commitIndex, from: sm.id}})   //hearbeat timeout
			}
			
		}

	case AppendEv:
		cmd := ev.(AppendEv)
		if sm.state == "follower"{
			actions = append(actions, AppendEv{data : cmd.data})  //forward to leader
		}else if sm.state == "leader"{
			actions = append(actions, LogStoreEv{index: sm.lastLogIndex+1, entry: logEntries{sm.term, cmd.data}})

		}

	case Send:


	case LogStoreEv:
		cmd := ev.(LogStoreEv)
		sm.log[cmd.index] = cmd.entry
	// other cases
	default: println ("Unrecognized")
	}
	return actions
}