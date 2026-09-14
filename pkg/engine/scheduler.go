package engine

import (
	"fmt"
	"time"

	"github.com/esrrhs/go_llm_engine/pkg/models"
)

// Scheduler governs the lifecycle, dependency resolution, and execution sequence of the TaskTree.
type Scheduler struct {
	tree *TaskTree
}

// NewScheduler creates a scheduler for the given tree.
func NewScheduler(tree *TaskTree) *Scheduler {
	return &Scheduler{tree: tree}
}

// GetTree returns the managed TaskTree.
func (s *Scheduler) GetTree() *TaskTree {
	return s.tree
}

// ValidateDependencies verifies that there are no missing or circular dependencies in the tree.
func (s *Scheduler) ValidateDependencies() error {
	s.tree.mu.RLock()
	defer s.tree.mu.RUnlock()

	// Check if all referenced dependencies exist
	for _, node := range s.tree.Nodes {
		for _, depID := range node.Contract.Dependencies {
			if _, exists := s.tree.Nodes[depID]; !exists {
				return fmt.Errorf("node %s references non-existent dependency %s", node.ID, depID)
			}
		}
	}

	// Cycle detection using DFS
	visited := make(map[string]int) // 0: unvisited, 1: visiting (in stack), 2: visited

	var dfs func(id string) error
	dfs = func(id string) error {
		visited[id] = 1
		node := s.tree.Nodes[id]
		for _, depID := range node.Contract.Dependencies {
			if visited[depID] == 1 {
				return fmt.Errorf("circular dependency detected involving %s and %s", id, depID)
			}
			if visited[depID] == 0 {
				if err := dfs(depID); err != nil {
					return err
				}
			}
		}
		visited[id] = 2
		return nil
	}

	for id := range s.tree.Nodes {
		if visited[id] == 0 {
			if err := dfs(id); err != nil {
				return err
			}
		}
	}

	return nil
}

// AreDependenciesSatisfied checks if all dependencies of the node are in COMPLETED state.
func (s *Scheduler) AreDependenciesSatisfied(node *models.TaskNode) bool {
	for _, depID := range node.Contract.Dependencies {
		depNode, exists := s.tree.GetNode(depID)
		if !exists || depNode.State != models.TaskStateCompleted {
			return false
		}
	}
	return true
}

// GetNextDecomposableNode finds a pending compound node ready to be broken down.
func (s *Scheduler) GetNextDecomposableNode() *models.TaskNode {
	s.tree.mu.RLock()
	defer s.tree.mu.RUnlock()

	for _, node := range s.tree.Nodes {
		if node.Type == models.NodeTypeCompound && node.State == models.TaskStatePending {
			// Check if it already has children generated
			if len(node.ChildrenIDs) == 0 && s.AreDependenciesSatisfied(node) {
				return node
			}
		}
	}
	return nil
}

// GetReadyLeafNodes returns all pending leaf nodes whose dependencies are satisfied.
func (s *Scheduler) GetReadyLeafNodes() []*models.TaskNode {
	s.tree.mu.RLock()
	defer s.tree.mu.RUnlock()

	ready := make([]*models.TaskNode, 0)
	for _, node := range s.tree.Nodes {
		if node.Type == models.NodeTypeLeaf && node.State == models.TaskStatePending {
			if s.AreDependenciesSatisfied(node) {
				ready = append(ready, node)
			}
		}
	}
	return ready
}

// UpdateNodeState updates the state of a node and automatically triggers parent state evaluation.
func (s *Scheduler) UpdateNodeState(nodeID string, newState models.TaskState, errorMsg string) error {
	s.tree.mu.Lock()
	node, exists := s.tree.Nodes[nodeID]
	if !exists {
		s.tree.mu.Unlock()
		return fmt.Errorf("node %s not found", nodeID)
	}

	node.State = newState
	node.ErrorMsg = errorMsg
	node.UpdatedAt = time.Now()
	s.tree.mu.Unlock()

	// Bubble up completion or failure check to parent
	if node.ParentID != "" {
		s.checkAndUpdateParent(node.ParentID)
	}
	return nil
}

// checkAndUpdateParent checks if all children of parent are completed, and if so marks parent completed.
func (s *Scheduler) checkAndUpdateParent(parentID string) {
	s.tree.mu.Lock()
	defer s.tree.mu.Unlock()

	parent, exists := s.tree.Nodes[parentID]
	if !exists || parent.Type != models.NodeTypeCompound {
		return
	}

	if len(parent.ChildrenIDs) == 0 {
		return
	}

	allCompleted := true
	hasFailed := false

	for _, cid := range parent.ChildrenIDs {
		child, exists := s.tree.Nodes[cid]
		if !exists {
			continue
		}
		if child.State == models.TaskStateFailed {
			hasFailed = true
		}
		if child.State != models.TaskStateCompleted {
			allCompleted = false
		}
	}

	if allCompleted {
		parent.State = models.TaskStateCompleted
		parent.UpdatedAt = time.Now()
	} else if hasFailed {
		// If any child failed permanently, parent reflects failure
		parent.State = models.TaskStateFailed
		parent.UpdatedAt = time.Now()
	} else {
		// If at least one child is running or decomposing, parent is running
		parent.State = models.TaskStateRunning
		parent.UpdatedAt = time.Now()
	}

	// Recursively bubble up to grandparent if needed
	if parent.ParentID != "" {
		go s.checkAndUpdateParent(parent.ParentID)
	}
}

// IsComplete returns true if the entire tree has successfully reached COMPLETED state.
func (s *Scheduler) IsComplete() bool {
	s.tree.mu.RLock()
	defer s.tree.mu.RUnlock()

	root, exists := s.tree.Nodes[s.tree.RootID]
	if !exists {
		return false
	}
	return root.State == models.TaskStateCompleted
}

// HasFailed returns true if root or any terminal failure prevents progress.
func (s *Scheduler) HasFailed() bool {
	s.tree.mu.RLock()
	defer s.tree.mu.RUnlock()

	root, exists := s.tree.Nodes[s.tree.RootID]
	if !exists {
		return true
	}
	return root.State == models.TaskStateFailed
}
