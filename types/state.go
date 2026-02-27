package types

import "github.com/ethereum/go-ethereum/common"

// State keeps track of address -> account mappings.
type State map[common.Address]*Account

// NewState copies an existing map into a dedicated State.
func NewState(initial map[common.Address]*Account) State {
	state := make(State, len(initial))
	for addr, acct := range initial {
		if acct == nil {
			continue
		}
		copyAcct := *acct
		state[addr] = &copyAcct
	}
	return state
}

// Clone returns a shallow copy so callers can run dry-runs safely.
func (s State) Clone() State {
	if s == nil {
		return make(State)
	}
	clone := make(State, len(s))
	for addr, acct := range s {
		if acct == nil {
			continue
		}
		copyAcct := *acct
		clone[addr] = &copyAcct
	}
	return clone
}
