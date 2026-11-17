// Package artifacts parses the headers and first lines of the v2 artifact
// contracts: intake.md, build.md, round checks.md, round review-<r>.md and
// decision.md. It reads no files: every function takes the content.
package artifacts

import (
	"fmt"
	"regexp"
	"strings"
)

// Invalid is the verdict of a review whose first line is out of contract.
const Invalid = "invalid"

var verdictPattern = regexp.MustCompile(`^VERDICT:\s*(PASS|CHANGES)$`)

// Intake is the header of intake.md.
type Intake struct {
	Decision string
	Risk     string
	Surfaces []string
	Rescoped bool
}

// Checks is the header of a round's checks.md.
type Checks struct {
	Gate   string
	Commit string
}

// Header returns the KEY: value lines at the top of content, up to the first
// blank line. Keys are trimmed and upper-cased, values trimmed.
func Header(content string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "" {
			break
		}
		key, value, _ := strings.Cut(line, ":")
		fields[strings.ToUpper(strings.TrimSpace(key))] = strings.TrimSpace(value)
	}
	return fields
}

// ParseIntake returns the header of intake.md, or an error when DECISION or
// RISK is out of contract.
func ParseIntake(content string) (Intake, error) {
	h := Header(content)
	decision := h["DECISION"]
	if !oneOf(decision, "ACCEPT", "NEEDS-INFO", "REJECT") {
		return Intake{}, fmt.Errorf("intake.md: DECISION %q is not ACCEPT, NEEDS-INFO or REJECT", decision)
	}
	risk := h["RISK"]
	if !oneOf(risk, "low", "medium", "high") {
		return Intake{}, fmt.Errorf("intake.md: RISK %q is not low, medium or high", risk)
	}
	surfaces := []string{}
	for _, item := range strings.Split(h["SURFACES"], ",") {
		item = strings.TrimSpace(item)
		if item != "" && item != "none" {
			surfaces = append(surfaces, item)
		}
	}
	return Intake{Decision: decision, Risk: risk, Surfaces: surfaces, Rescoped: h["RESCOPED"] == "yes"}, nil
}

// ParseBuild returns READY or BLOCKED from the first line of build.md.
func ParseBuild(content string) (string, error) {
	line := firstLine(content)
	switch line {
	case "STATUS: READY":
		return "READY", nil
	case "STATUS: BLOCKED":
		return "BLOCKED", nil
	}
	return "", fmt.Errorf("build.md: first line %q is not STATUS: READY or STATUS: BLOCKED", line)
}

// ParseChecks returns the header of a round's checks.md, or an error when
// GATE is out of contract. COMMIT is returned as written.
func ParseChecks(content string) (Checks, error) {
	h := Header(content)
	gate := h["GATE"]
	if !oneOf(gate, "PASS", "FAIL") {
		return Checks{}, fmt.Errorf("checks.md: GATE %q is not PASS or FAIL", gate)
	}
	return Checks{Gate: gate, Commit: h["COMMIT"]}, nil
}

// ParseReview returns PASS or CHANGES from the first line of a review, or
// Invalid for any other first line.
func ParseReview(content string) string {
	m := verdictPattern.FindStringSubmatch(firstLine(content))
	if m == nil {
		return Invalid
	}
	return m[1]
}

// ParseDecision returns APPROVE or ESCALATE from the first line of
// decision.md.
func ParseDecision(content string) (string, error) {
	line := firstLine(content)
	if !oneOf(line, "APPROVE", "ESCALATE") {
		return "", fmt.Errorf("decision.md: first line %q is not APPROVE or ESCALATE", line)
	}
	return line, nil
}

func firstLine(content string) string {
	line, _, _ := strings.Cut(content, "\n")
	return strings.TrimSpace(line)
}

func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}
