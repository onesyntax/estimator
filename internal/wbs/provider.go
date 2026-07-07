package wbs

// Provider turns a requirement into an ordered list of task descriptions.
type Provider interface {
	Generate(requirement string) ([]string, error)
}

// Primer is implemented by providers whose upcoming outputs can be seeded ahead
// of generation. It lets the service seed a deterministic provider without
// depending on any concrete provider type.
type Primer interface {
	Prime(tasks []string)
}

// PrimedProvider is a deterministic Provider whose output is primed ahead of
// each generation. Primings are consumed one-shot in FIFO order, so successive
// generations stay isolated from one another.
type PrimedProvider struct {
	queue [][]string
}

// Prime enqueues the exact ordered task list the next generation will return.
func (p *PrimedProvider) Prime(tasks []string) {
	primed := make([]string, len(tasks))
	copy(primed, tasks)
	p.queue = append(p.queue, primed)
}

// Generate returns the next primed task list, consuming it. When nothing is
// primed it returns an empty list.
func (p *PrimedProvider) Generate(requirement string) ([]string, error) {
	if len(p.queue) == 0 {
		return []string{}, nil
	}
	next := p.queue[0]
	p.queue = p.queue[1:]
	return next, nil
}
