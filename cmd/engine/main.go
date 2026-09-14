package main

import (
	"fmt"

	"github.com/esrrhs/go_llm_engine/pkg/engine"
	"github.com/esrrhs/go_llm_engine/pkg/models"
)

func main() {
	fmt.Println("=== go_llm_engine: Task Tree Engine initialized ===")

	tree := engine.NewTaskTree("demo_session", "Target: Create Minimal HTTP Service", "End-to-end task decomposition demo")
	c1, _ := tree.AddChild("root", "task_scaffold", "Project Scaffolding", "Initialize directory and go.mod", models.NodeTypeLeaf)
	c2, _ := tree.AddChild("root", "task_handler", "Write Handler", "Create health check handler", models.NodeTypeLeaf)
	c2.Contract.Dependencies = append(c2.Contract.Dependencies, c1.ID)

	fmt.Println("\nInitial Tree Structure:")
	fmt.Print(tree.RenderVisualTree())

	storage, _ := engine.NewStorage(".go_llm_engine")
	if err := storage.SaveTree(tree); err == nil {
		fmt.Printf("\nSaved tree session to: %s\n", storage.GetTreeFilePath(tree.ID))
	}
}
