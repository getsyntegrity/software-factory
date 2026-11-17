// Package artifacts parses the first-line and header contracts of the v2
// job artifacts: intake.md, build.md, round checks.md, review-<r>.md and
// decision.md. It reads only the content its caller passes in; it does no
// disk access. An out-of-contract value is reported as an error naming the
// file, never turned into a valid one; a review instead gets the verdict
// invalid.
package artifacts
