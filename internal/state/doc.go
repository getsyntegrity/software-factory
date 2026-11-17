// Package state is the pure job state machine: the job states, the legal
// transitions between them and the guards each move must pass. It reads only
// the facts its caller passes in; it does no storage, agent, git or GitHub
// work.
package state
