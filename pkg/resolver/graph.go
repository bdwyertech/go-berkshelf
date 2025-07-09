package resolver

import (
	"fmt"

	"github.com/bdwyertech/go-berkshelf/pkg/berkshelf"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

// DependencyGraph represents cookbook dependencies using gonum's graph
type DependencyGraph struct {
	graph     *simple.DirectedGraph
	nodes     map[string]*CookbookNode
	nodesByID map[int64]*CookbookNode
	nextID    int64
}

// CookbookNode represents a cookbook in the dependency graph
type CookbookNode struct {
	id       int64
	Name     string
	Version  *berkshelf.Version
	Cookbook *berkshelf.Cookbook
	Resolved bool
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		graph:     simple.NewDirectedGraph(),
		nodes:     make(map[string]*CookbookNode),
		nodesByID: make(map[int64]*CookbookNode),
		nextID:    1,
	}
}

// ID implements graph.Node interface
func (n *CookbookNode) ID() int64 {
	return n.id
}

// String returns a string representation of the cookbook node
func (n *CookbookNode) String() string {
	if n.Version != nil {
		return fmt.Sprintf("%s (%s)", n.Name, n.Version.String())
	}
	return n.Name
}

// AddCookbook adds a cookbook to the graph
func (g *DependencyGraph) AddCookbook(cookbook *berkshelf.Cookbook) *CookbookNode {
	key := cookbook.Name
	if existingNode, exists := g.nodes[key]; exists {
		// Update existing node
		existingNode.Cookbook = cookbook
		existingNode.Version = cookbook.Version
		return existingNode
	}

	// Create new node
	node := &CookbookNode{
		id:       g.nextID,
		Name:     cookbook.Name,
		Version:  cookbook.Version,
		Cookbook: cookbook,
		Resolved: false,
	}
	g.nextID++

	// Add to graph and mappings
	g.graph.AddNode(node)
	g.nodes[key] = node
	g.nodesByID[node.id] = node

	return node
}

// GetCookbook retrieves a cookbook node by name
func (g *DependencyGraph) GetCookbook(name string) (*CookbookNode, bool) {
	node, exists := g.nodes[name]
	return node, exists
}

// AddDependency adds a dependency edge between cookbooks
func (g *DependencyGraph) AddDependency(from, to *CookbookNode, constraint *berkshelf.Constraint) {
	if from == nil || to == nil {
		return
	}

	// Add nodes to graph if not already present
	if g.graph.Node(from.ID()) == nil {
		g.graph.AddNode(from)
	}
	if g.graph.Node(to.ID()) == nil {
		g.graph.AddNode(to)
	}

	// Add edge
	edge := g.graph.NewEdge(from, to)
	g.graph.SetEdge(edge)
}

// HasDependency checks if a dependency exists between two cookbooks
func (g *DependencyGraph) HasDependency(from, to *CookbookNode) bool {
	if from == nil || to == nil {
		return false
	}
	return g.graph.HasEdgeFromTo(from.ID(), to.ID())
}

// GetDependencies returns all direct dependencies of a cookbook
func (g *DependencyGraph) GetDependencies(node *CookbookNode) []*CookbookNode {
	if node == nil {
		return nil
	}

	var deps []*CookbookNode
	it := g.graph.From(node.ID())
	for it.Next() {
		if depNode, exists := g.nodesByID[it.Node().ID()]; exists {
			deps = append(deps, depNode)
		}
	}
	return deps
}

// GetDependents returns all cookbooks that depend on the given cookbook
func (g *DependencyGraph) GetDependents(node *CookbookNode) []*CookbookNode {
	if node == nil {
		return nil
	}

	var dependents []*CookbookNode
	it := g.graph.To(node.ID())
	for it.Next() {
		if depNode, exists := g.nodesByID[it.Node().ID()]; exists {
			dependents = append(dependents, depNode)
		}
	}
	return dependents
}

// TopologicalSort returns cookbooks in dependency order
func (g *DependencyGraph) TopologicalSort() ([]*CookbookNode, error) {
	// Use gonum's topological sort
	sorted, err := topo.Sort(g.graph)
	if err != nil {
		return nil, fmt.Errorf("dependency cycle detected: %w", err)
	}

	// Convert to cookbook nodes
	var result []*CookbookNode
	for _, node := range sorted {
		if cookbookNode, exists := g.nodesByID[node.ID()]; exists {
			result = append(result, cookbookNode)
		}
	}

	return result, nil
}

// HasCycles checks if the dependency graph has circular dependencies
func (g *DependencyGraph) HasCycles() bool {
	_, err := topo.Sort(g.graph)
	return err != nil
}

// GetCycles returns any circular dependencies found in the graph
func (g *DependencyGraph) GetCycles() [][]string {
	if !g.HasCycles() {
		return nil
	}

	// Simple cycle detection - this could be enhanced
	var cycles [][]string

	// For now, just return an indication that cycles exist
	// A more sophisticated implementation would return the actual cycles
	if g.HasCycles() {
		cycles = append(cycles, []string{"cycle detected - detailed cycle analysis not yet implemented"})
	}

	return cycles
}

// NodeCount returns the number of cookbooks in the graph
func (g *DependencyGraph) NodeCount() int {
	return len(g.nodes)
}

// EdgeCount returns the number of dependency relationships in the graph
func (g *DependencyGraph) EdgeCount() int {
	var count int
	nodes := g.graph.Nodes()
	for nodes.Next() {
		from := nodes.Node()
		to := g.graph.From(from.ID())
		for to.Next() {
			count++
		}
	}
	return count
}

// AllCookbooks returns all cookbook nodes in the graph
func (g *DependencyGraph) AllCookbooks() []*CookbookNode {
	var cookbooks []*CookbookNode
	for _, node := range g.nodes {
		cookbooks = append(cookbooks, node)
	}
	return cookbooks
}

// Clone creates a deep copy of the dependency graph
func (g *DependencyGraph) Clone() *DependencyGraph {
	clone := NewDependencyGraph()

	// Copy all nodes
	for _, node := range g.nodes {
		clonedNode := &CookbookNode{
			id:       node.id,
			Name:     node.Name,
			Version:  node.Version,
			Cookbook: node.Cookbook, // Shallow copy of cookbook
			Resolved: node.Resolved,
		}
		clone.graph.AddNode(clonedNode)
		clone.nodes[node.Name] = clonedNode
		clone.nodesByID[node.id] = clonedNode
	}

	// Copy all edges
	nodes := g.graph.Nodes()
	for nodes.Next() {
		from := nodes.Node()
		to := g.graph.From(from.ID())
		for to.Next() {
			edge := g.graph.NewEdge(clone.nodesByID[from.ID()], clone.nodesByID[to.Node().ID()])
			clone.graph.SetEdge(edge)
		}
	}

	clone.nextID = g.nextID
	return clone
}
