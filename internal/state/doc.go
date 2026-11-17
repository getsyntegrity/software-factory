// Package state is the pure job state machine: the job states, the legal
// transitions between them and the guards each move must pass. It reads only
// the facts its caller passes in; it does no storage, agent, git or GitHub
// work.
//
// It has two profiles. V2 keeps factory v2 compatibility. ODD is V2 plus
// returns to spec from build, gate and review, a round consumed by every
// build, and candidate-bound evidence: approve and merging accept only a
// gate, reviews and a decision produced for the current candidate.
package state
