package core

import "sync"

type State struct {
	rw      sync.Mutex
	require string
	forced  string
	status  map[string]string // map[node]status
}

func NewState() *State {
	return &State{
		status: make(map[string]string),
	}
}

func (s *State) AddMaster(uuid string) *State {
	return nil
}

func (s *State) AnyOf() {
	s.require = "any"
}

func (s *State) AllOf() {
	s.require = "all"
}

func (s *State) Force(state string) {
	s.forced = state
}

func (s *State) UnForce() {
	s.forced = ""
}

func (s *State) AddClient(masteruuid, clientuuid string) {
	s.rw.Lock()
	defer s.rw.Unlock()
	s.status[masteruuid] = clientuuid
}
