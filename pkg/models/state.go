package models

// TaskState represents the lifecycle state of a task node in the tree.
type TaskState string

const (
	TaskStatePending     TaskState = "PENDING"
	TaskStateDecomposing TaskState = "DECOMPOSING"
	TaskStateRunning     TaskState = "RUNNING"
	TaskStateVerifying   TaskState = "VERIFYING"
	TaskStateCompleted   TaskState = "COMPLETED"
	TaskStateFailed      TaskState = "FAILED"
	TaskStateSkipped     TaskState = "SKIPPED"
)

// NodeType indicates whether the node is a compound task or an atomic leaf task.
type NodeType string

const (
	// NodeTypeCompound represents a high-level goal that needs decomposition.
	NodeTypeCompound NodeType = "COMPOUND"
	// NodeTypeLeaf represents an atomic, isolated task ready for single-step execution.
	NodeTypeLeaf NodeType = "LEAF"
)

// IsTerminal returns true if the task state is in a terminal condition.
func (s TaskState) IsTerminal() bool {
	return s == TaskStateCompleted || s == TaskStateFailed || s == TaskStateSkipped
}
