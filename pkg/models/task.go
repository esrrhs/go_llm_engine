package models

import (
	"time"
)

// TaskNode represents a single unit of goal/work in the decomposition tree.
type TaskNode struct {
	ID          string       `json:"id"`
	ParentID    string       `json:"parent_id,omitempty"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Type        NodeType     `json:"type"`
	State       TaskState    `json:"state"`
	Depth       int          `json:"depth"`
	ChildrenIDs []string     `json:"children_ids,omitempty"`
	Contract    ContractSpec `json:"contract"`
	DoD         DoD          `json:"dod"`

	// Execution & retry tracking
	RetryCount    int       `json:"retry_count"`
	MaxRetries    int       `json:"max_retries"`
	ErrorMsg      string    `json:"error_msg,omitempty"`
	ResultSummary string    `json:"result_summary,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewTaskNode initializes a new task node with sensible defaults.
func NewTaskNode(id, parentID, title, desc string, nodeType NodeType, depth int) *TaskNode {
	now := time.Now()
	return &TaskNode{
		ID:          id,
		ParentID:    parentID,
		Title:       title,
		Description: desc,
		Type:        nodeType,
		State:       TaskStatePending,
		Depth:       depth,
		ChildrenIDs: make([]string, 0),
		Contract: ContractSpec{
			Inputs:       make([]string, 0),
			Outputs:      make([]string, 0),
			Dependencies: make([]string, 0),
			Constraints:  make([]string, 0),
		},
		DoD: DoD{
			Commands:   make([]string, 0),
			TimeoutSec: 60,
		},
		MaxRetries: 3,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// IsLeaf returns true if node is a leaf task.
func (n *TaskNode) IsLeaf() bool {
	return n.Type == NodeTypeLeaf
}

// CanRetry returns whether node can still be retried.
func (n *TaskNode) CanRetry() bool {
	return n.RetryCount < n.MaxRetries
}
