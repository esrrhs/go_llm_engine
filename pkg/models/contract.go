package models

// ContractSpec defines the interface, input/output, and contextual constraints of a task.
type ContractSpec struct {
	// Inputs specifies required files, functions, variables, or data dependencies.
	Inputs []string `json:"inputs,omitempty"`
	// Outputs specifies target files, function signatures, data structures to be produced.
	Outputs []string `json:"outputs,omitempty"`
	// Dependencies contains TaskNode IDs that must be completed before this task can execute.
	Dependencies []string `json:"dependencies,omitempty"`
	// Constraints specifies strict boundaries (e.g. "Do not modify file X", "Must use Go standard library only").
	Constraints []string `json:"constraints,omitempty"`
}

// DoD represents the Definition of Done (completion criteria and verification instructions).
type DoD struct {
	// Description provides human-readable acceptance criteria.
	Description string `json:"description"`
	// Commands are automated shell commands executed to verify completion (e.g., "go test -v ./...", "go build").
	Commands []string `json:"commands,omitempty"`
	// ExpectedOutput is an optional string pattern that command output must match or contain.
	ExpectedOutput string `json:"expected_output,omitempty"`
	// TimeoutSec specifies max allowed execution time for verification commands in seconds. Default 60s.
	TimeoutSec int `json:"timeout_sec,omitempty"`
}
