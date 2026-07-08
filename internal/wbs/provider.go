package wbs

// Provider turns a requirement into an ordered list of task descriptions.
type Provider interface {
	Generate(req Requirement) ([]string, error)
}

// Primer is implemented by providers whose upcoming outputs can be seeded ahead
// of generation. It lets the service seed a deterministic provider without
// depending on any concrete provider type.
type Primer interface {
	Prime(tasks []string)
}

// RiskProvider flags the risks of an approved WBS, returning one risk
// assignment per risk it identifies for a task.
type RiskProvider interface {
	FlagRisks(tasks []Task) ([]RiskAssignment, error)
}

// RiskPrimer is implemented by risk providers whose next output can be seeded
// ahead of flagging, so the service can seed a deterministic risk provider
// without depending on any concrete provider type.
type RiskPrimer interface {
	PrimeRisks(assignments []RiskAssignment)
}

// PrimedRiskProvider is a deterministic RiskProvider whose output is primed
// ahead of each flag. Primings are consumed one-shot in FIFO order, so
// successive flags stay isolated from one another.
type PrimedRiskProvider struct {
	queue [][]RiskAssignment
}

// PrimeRisks enqueues the exact risk assignments the next flag will return.
func (p *PrimedRiskProvider) PrimeRisks(assignments []RiskAssignment) {
	primed := make([]RiskAssignment, len(assignments))
	copy(primed, assignments)
	p.queue = append(p.queue, primed)
}

// FlagRisks returns the next primed assignment list, consuming it. The tasks are
// ignored: priming makes output deterministic. When nothing is primed it returns
// no assignments.
func (p *PrimedRiskProvider) FlagRisks(tasks []Task) ([]RiskAssignment, error) {
	if len(p.queue) == 0 {
		return nil, nil
	}
	next := p.queue[0]
	p.queue = p.queue[1:]
	return next, nil
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

// Generate returns the next primed task list, consuming it. The requirement is
// ignored: priming makes output deterministic. When nothing is primed it returns
// an empty list.
func (p *PrimedProvider) Generate(req Requirement) ([]string, error) {
	if len(p.queue) == 0 {
		return []string{}, nil
	}
	next := p.queue[0]
	p.queue = p.queue[1:]
	return next, nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-07T22:19:05+05:30","module_hash":"eb62bc9443091a3d4a71f454204169652e5d4eee960212f8c63bb6a2d524c182","functions":[{"id":"func/PrimedProvider.Prime","name":"PrimedProvider.Prime","line":23,"end_line":27,"hash":"2f92a980d1e186da1336bb88bac0eae14040896117a080e186f2eacfe74d0cda"},{"id":"func/PrimedProvider.Generate","name":"PrimedProvider.Generate","line":31,"end_line":38,"hash":"8f1659f094b77870fc5e2f9b32f66559d9498ad3d8384bc797342485934a2436"}]}
// mutate4go-manifest-end
