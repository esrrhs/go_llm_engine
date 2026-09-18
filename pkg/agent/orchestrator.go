package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/esrrhs/go_llm_engine/pkg/engine"
	"github.com/esrrhs/go_llm_engine/pkg/llm"
	"github.com/esrrhs/go_llm_engine/pkg/models"
	"github.com/esrrhs/go_llm_engine/pkg/tools"
)

// Orchestrator is the top-level decompose → execute → verify loop.
type Orchestrator struct {
	cfg     Config
	tree    *engine.TaskTree
	sched   *engine.Scheduler
	storage *engine.Storage
	llm     llm.Client
	sandbox *tools.Sandbox
	log     *Logger
}

// New creates an orchestrator around an existing tree.
func New(cfg Config, tree *engine.TaskTree, client llm.Client, log *Logger) (*Orchestrator, error) {
	if log == nil {
		log = NewLogger(cfg.Verbose)
	}
	sandbox, err := tools.NewSandbox(cfg.WorkDir)
	if err != nil {
		return nil, err
	}
	storage, err := engine.NewStorage(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	tree.WorkDir = sandbox.Root
	tree.Goal = cfg.Goal
	return &Orchestrator{
		cfg:     cfg,
		tree:    tree,
		sched:   engine.NewScheduler(tree),
		storage: storage,
		llm:     client,
		sandbox: sandbox,
		log:     log,
	}, nil
}

// NewFromGoal starts a fresh session.
func NewFromGoal(cfg Config, client llm.Client, log *Logger) (*Orchestrator, error) {
	if cfg.SessionID == "" {
		cfg.SessionID = "sess_" + time.Now().Format("20060102_150405")
	}
	if cfg.Goal == "" {
		return nil, fmt.Errorf("goal is required")
	}
	tree := engine.NewTaskTree(cfg.SessionID, cfg.Goal, cfg.Goal)
	o, err := New(cfg, tree, client, log)
	if err != nil {
		return nil, err
	}
	_ = o.tree.UpdateNode(tree.RootID, func(n *models.TaskNode) error {
		n.MaxRetries = cfg.MaxRetries
		return nil
	})
	return o, nil
}

// Load resumes a saved session.
func Load(cfg Config, client llm.Client, log *Logger) (*Orchestrator, error) {
	storage, err := engine.NewStorage(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	if cfg.SessionID == "" {
		latest, err := os.ReadFile(filepath.Join(cfg.DataDir, "LATEST"))
		if err != nil {
			return nil, fmt.Errorf("no session specified and no LATEST pointer: %w", err)
		}
		cfg.SessionID = string(bytesTrim(latest))
	}
	tree, err := storage.LoadTree(cfg.SessionID)
	if err != nil {
		return nil, err
	}
	if cfg.Goal == "" {
		cfg.Goal = tree.Goal
	}
	if cfg.WorkDir == "." && tree.WorkDir != "" {
		cfg.WorkDir = tree.WorkDir
	}
	o, err := New(cfg, tree, client, log)
	if err != nil {
		return nil, err
	}
	o.resetInterrupted()
	return o, nil
}

func bytesTrim(b []byte) string {
	s := string(b)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func (o *Orchestrator) resetInterrupted() {
	ids := o.tree.NodeIDsInStates(
		models.TaskStateRunning,
		models.TaskStateDecomposing,
		models.TaskStateVerifying,
	)
	for _, id := range ids {
		_ = o.sched.UpdateNodeState(id, models.TaskStatePending, "interrupted; will retry")
	}
}

// SessionID returns the tree session id.
func (o *Orchestrator) SessionID() string { return o.tree.ID }

// Tree returns the live task tree.
func (o *Orchestrator) Tree() *engine.TaskTree { return o.tree }

// StorageDir is the persistence directory.
func (o *Orchestrator) StorageDir() string { return o.cfg.DataDir }

// Run drives the engine until the root completes, fails, or the context is cancelled.
func (o *Orchestrator) Run(ctx context.Context) error {
	o.log.Banner("go_llm_engine")
	o.log.Infof("session %s", o.tree.ID)
	o.log.Infof("model   %s", o.cfg.Model)
	o.log.Infof("workdir %s", o.sandbox.Root)
	o.log.Infof("goal    %s", o.cfg.Goal)
	o.printTree()

	if err := o.checkpoint(); err != nil {
		return err
	}

	idle := 0
	for {
		if err := ctx.Err(); err != nil {
			_ = o.checkpoint()
			return err
		}
		if o.sched.IsComplete() {
			o.printTree()
			o.log.Okf("root task completed")
			return o.checkpoint()
		}
		if o.sched.HasFailed() {
			root := o.tree.GetRoot()
			msg := ""
			if root != nil {
				msg = root.ErrorMsg
			}
			o.printTree()
			return fmt.Errorf("root task failed: %s", msg)
		}

		if node := o.sched.GetNextDecomposableNode(); node != nil {
			idle = 0
			if err := o.decompose(ctx, node); err != nil {
				o.log.Errorf("decompose %s: %v", node.ID, err)
				_ = o.sched.UpdateNodeState(node.ID, models.TaskStateFailed, err.Error())
			}
			_ = o.checkpoint()
			o.printTree()
			continue
		}

		ready := o.sched.GetReadyLeafNodes()
		if len(ready) == 0 {
			idle++
			if idle > 3 {
				return fmt.Errorf("no runnable tasks (%s)", o.sched.DescribeStuck())
			}
			time.Sleep(50 * time.Millisecond)
			continue
		}
		idle = 0
		leaf := ready[0]
		if err := o.executeLeaf(ctx, leaf); err != nil {
			return err
		}
		_ = o.checkpoint()
		o.printTree()
	}
}

func (o *Orchestrator) executeLeaf(ctx context.Context, leaf *models.TaskNode) error {
	prevErr := ""
	for attempt := 1; ; attempt++ {
		live, ok := o.tree.CloneNode(leaf.ID)
		if !ok {
			return fmt.Errorf("leaf %s disappeared", leaf.ID)
		}
		leaf = live

		o.log.Infof("execute %s [attempt %d]: %s", leaf.ID, attempt, leaf.Title)
		if err := o.sched.UpdateNodeState(leaf.ID, models.TaskStateRunning, ""); err != nil {
			return err
		}

		summary, err := o.runWorker(ctx, leaf, prevErr)
		if err != nil {
			if ctx.Err() != nil {
				_ = o.sched.UpdateNodeState(leaf.ID, models.TaskStatePending, "interrupted")
				return ctx.Err()
			}
			prevErr = err.Error()
			o.log.Warnf("worker error: %s", prevErr)
			_ = o.tree.UpdateNode(leaf.ID, func(n *models.TaskNode) error {
				n.RetryCount++
				n.ErrorMsg = prevErr
				return nil
			})
			if err := o.retryOrGiveUp(ctx, leaf, attempt, prevErr); err != nil {
				return err
			}
			continue
		}

		if err := o.sched.UpdateNodeState(leaf.ID, models.TaskStateVerifying, ""); err != nil {
			return err
		}
		vr := o.verify(ctx, leaf)
		if ctx.Err() != nil {
			_ = o.sched.UpdateNodeState(leaf.ID, models.TaskStatePending, "interrupted")
			return ctx.Err()
		}
		if vr.OK {
			_ = o.tree.UpdateNode(leaf.ID, func(n *models.TaskNode) error {
				n.ResultSummary = summary
				n.ErrorMsg = ""
				return nil
			})
			if err := o.sched.UpdateNodeState(leaf.ID, models.TaskStateCompleted, ""); err != nil {
				return err
			}
			o.log.Okf("completed %s — %s", leaf.ID, truncate(summary, 120))
			return nil
		}

		prevErr = vr.Output
		_ = o.tree.UpdateNode(leaf.ID, func(n *models.TaskNode) error {
			n.RetryCount++
			n.ErrorMsg = vr.Output
			return nil
		})
		o.log.Warnf("verification failed for %s (attempt %d)", leaf.ID, attempt)
		if err := o.retryOrGiveUp(ctx, leaf, attempt, vr.Output); err != nil {
			return err
		}
	}
}

func (o *Orchestrator) retryOrGiveUp(ctx context.Context, leaf *models.TaskNode, attempt int, errMsg string) error {
	if o.cfg.MaxRetries > 0 && attempt >= o.cfg.MaxRetries {
		return o.handleLeafFailure(leaf, errMsg)
	}
	if err := o.sched.UpdateNodeState(leaf.ID, models.TaskStatePending, errMsg); err != nil {
		return err
	}
	_ = o.checkpoint()
	delay := RetryDelay(attempt, o.cfg.retryMin(), o.cfg.retryMax())
	o.log.Warnf("retry %s in %s", leaf.ID, delay)
	if err := waitBackoff(ctx, delay); err != nil {
		_ = o.sched.UpdateNodeState(leaf.ID, models.TaskStatePending, "interrupted")
		return err
	}
	return nil
}

func (o *Orchestrator) handleLeafFailure(leaf *models.TaskNode, errMsg string) error {
	live, ok := o.tree.CloneNode(leaf.ID)
	if ok {
		leaf = live
	}
	if leaf.DecomposeCount >= o.cfg.MaxRedecompose || leaf.Depth >= o.cfg.MaxDepth {
		o.log.Errorf("giving up on %s", leaf.ID)
		if err := o.sched.UpdateNodeState(leaf.ID, models.TaskStateFailed, errMsg); err != nil {
			return err
		}
		if o.sched.HasFailed() {
			return fmt.Errorf("task %s failed: %s", leaf.ID, truncate(errMsg, 500))
		}
		return nil
	}

	o.log.Warnf("re-splitting failed leaf %s", leaf.ID)
	err := o.tree.UpdateNode(leaf.ID, func(n *models.TaskNode) error {
		n.Type = models.NodeTypeCompound
		n.State = models.TaskStatePending
		n.DecomposeCount++
		n.RetryCount = 0
		n.ErrorMsg = errMsg
		n.ChildrenIDs = n.ChildrenIDs[:0]
		n.Contract.Constraints = append(n.Contract.Constraints,
			"Previous leaf execution failed; split into smaller tasks. Error: "+truncate(errMsg, 1500))
		return nil
	})
	if err != nil {
		return err
	}
	o.sched.RefreshAncestors(leaf.ID)
	return nil
}

func (o *Orchestrator) checkpoint() error {
	if err := o.storage.SaveTree(o.tree); err != nil {
		return err
	}
	latest := filepath.Join(o.cfg.DataDir, "LATEST")
	_ = os.WriteFile(latest, []byte(o.tree.ID+"\n"), 0644)
	return nil
}

func (o *Orchestrator) printTree() {
	done, total, pct := o.tree.GetLeafProgress()
	allDone, allTotal, allPct := o.tree.GetProgress()
	o.log.Infof("leaves %d/%d (%.0f%%)  nodes %d/%d (%.0f%%)", done, total, pct, allDone, allTotal, allPct)
	o.log.Print(o.tree.RenderVisualTree())
}

// Status prints the current tree without running.
func (o *Orchestrator) Status() {
	o.printTree()
	o.log.Infof("session file: %s", o.storage.GetTreeFilePath(o.tree.ID))
}
