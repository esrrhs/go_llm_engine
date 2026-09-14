package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/esrrhs/go_llm_engine/pkg/models"
)

// TaskTree coordinates the hierarchical structure of task nodes.
type TaskTree struct {
	ID        string                      `json:"id"`
	RootID    string                      `json:"root_id"`
	Nodes     map[string]*models.TaskNode `json:"nodes"`
	CreatedAt time.Time                   `json:"created_at"`
	UpdatedAt time.Time                   `json:"updated_at"`

	mu sync.RWMutex
}

// NewTaskTree creates a new tree with a root compound node.
func NewTaskTree(id, rootTitle, rootDesc string) *TaskTree {
	rootID := "root"
	rootNode := models.NewTaskNode(rootID, "", rootTitle, rootDesc, models.NodeTypeCompound, 0)

	tree := &TaskTree{
		ID:        id,
		RootID:    rootID,
		Nodes:     make(map[string]*models.TaskNode),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	tree.Nodes[rootID] = rootNode
	return tree
}

// AddNode adds an existing node to the tree and links it to its parent.
func (t *TaskTree) AddNode(node *models.TaskNode) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.Nodes[node.ID]; exists {
		return fmt.Errorf("node with id %s already exists", node.ID)
	}

	if node.ParentID != "" {
		parent, exists := t.Nodes[node.ParentID]
		if !exists {
			return fmt.Errorf("parent node %s does not exist", node.ParentID)
		}
		parent.ChildrenIDs = append(parent.ChildrenIDs, node.ID)
		parent.UpdatedAt = time.Now()
	}

	t.Nodes[node.ID] = node
	t.UpdatedAt = time.Now()
	return nil
}

// AddChild creates a new child node under parentID and attaches it.
func (t *TaskTree) AddChild(parentID, childID, title, desc string, nodeType models.NodeType) (*models.TaskNode, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	parent, exists := t.Nodes[parentID]
	if !exists {
		return nil, fmt.Errorf("parent node %s does not exist", parentID)
	}

	if _, exists := t.Nodes[childID]; exists {
		return nil, fmt.Errorf("node %s already exists", childID)
	}

	child := models.NewTaskNode(childID, parentID, title, desc, nodeType, parent.Depth+1)
	parent.ChildrenIDs = append(parent.ChildrenIDs, childID)
	parent.UpdatedAt = time.Now()

	t.Nodes[childID] = child
	t.UpdatedAt = time.Now()
	return child, nil
}

// GetNode safely retrieves a node by ID.
func (t *TaskTree) GetNode(id string) (*models.TaskNode, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	node, exists := t.Nodes[id]
	return node, exists
}

// GetRoot returns the root task node.
func (t *TaskTree) GetRoot() *models.TaskNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.Nodes[t.RootID]
}

// GetChildren returns all direct child nodes of given parentID.
func (t *TaskTree) GetChildren(parentID string) ([]*models.TaskNode, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	parent, exists := t.Nodes[parentID]
	if !exists {
		return nil, fmt.Errorf("node %s does not exist", parentID)
	}

	children := make([]*models.TaskNode, 0, len(parent.ChildrenIDs))
	for _, cid := range parent.ChildrenIDs {
		if child, ok := t.Nodes[cid]; ok {
			children = append(children, child)
		}
	}
	return children, nil
}

// GetProgress returns (completedCount, totalCount, percentage)
func (t *TaskTree) GetProgress() (int, int, float64) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	total := len(t.Nodes)
	if total == 0 {
		return 0, 0, 0.0
	}

	completed := 0
	for _, node := range t.Nodes {
		if node.State == models.TaskStateCompleted {
			completed++
		}
	}

	percent := (float64(completed) / float64(total)) * 100.0
	return completed, total, percent
}

// RenderVisualTree produces a clean CLI tree visualization.
func (t *TaskTree) RenderVisualTree() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var sb strings.Builder
	root, ok := t.Nodes[t.RootID]
	if !ok {
		return "[Empty Tree]"
	}

	t.renderNodeRecursive(&sb, root, "", true)
	return sb.String()
}

func (t *TaskTree) renderNodeRecursive(sb *strings.Builder, node *models.TaskNode, prefix string, isLast bool) {
	marker := "├── "
	if isLast {
		marker = "└── "
	}
	if node.ID == t.RootID {
		sb.WriteString(fmt.Sprintf("[%s] %s (Root: %s)\n", node.State, node.Title, node.ID))
	} else {
		sb.WriteString(fmt.Sprintf("%s%s[%s] %s (%s: %s)\n", prefix, marker, node.State, node.Title, node.Type, node.ID))
	}

	childPrefix := prefix
	if node.ID != t.RootID {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	numChildren := len(node.ChildrenIDs)
	for i, cid := range node.ChildrenIDs {
		child, ok := t.Nodes[cid]
		if ok {
			t.renderNodeRecursive(sb, child, childPrefix, i == numChildren-1)
		}
	}
}

// ToJSON serializes the task tree to JSON.
func (t *TaskTree) ToJSON() ([]byte, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return json.MarshalIndent(t, "", "  ")
}

// FromJSON deserializes JSON bytes into a TaskTree.
func FromJSON(data []byte) (*TaskTree, error) {
	var tree TaskTree
	if err := json.Unmarshal(data, &tree); err != nil {
		return nil, err
	}
	return &tree, nil
}
