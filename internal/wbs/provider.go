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
	p.queue = enqueueCopy(p.queue, assignments)
}

// FlagRisks returns the next primed assignment list, consuming it. The tasks are
// ignored: priming makes output deterministic. When nothing is primed it returns
// no assignments.
func (p *PrimedRiskProvider) FlagRisks(tasks []Task) ([]RiskAssignment, error) {
	return dequeue(&p.queue), nil
}

// EstimateProvider produces a 3-point estimate for each task of an approved WBS.
type EstimateProvider interface {
	Estimate(tasks []Task) ([]EstimateAssignment, error)
}

// EstimatePrimer is implemented by estimate providers whose next output can be
// seeded ahead of generation, so the service can seed a deterministic estimate
// provider without depending on any concrete provider type.
type EstimatePrimer interface {
	PrimeEstimates(assignments []EstimateAssignment)
}

// PrimedEstimateProvider is a deterministic EstimateProvider whose output is
// primed ahead of each generation. Primings are consumed one-shot in FIFO
// order, so successive generations stay isolated from one another.
type PrimedEstimateProvider struct {
	queue [][]EstimateAssignment
}

// PrimeEstimates enqueues the exact estimate assignments the next generation
// will return.
func (p *PrimedEstimateProvider) PrimeEstimates(assignments []EstimateAssignment) {
	p.queue = enqueueCopy(p.queue, assignments)
}

// Estimate returns the next primed assignment list, consuming it. The tasks are
// ignored: priming makes output deterministic. When nothing is primed it returns
// no assignments.
func (p *PrimedEstimateProvider) Estimate(tasks []Task) ([]EstimateAssignment, error) {
	return dequeue(&p.queue), nil
}

// PrimedProvider is a deterministic Provider whose output is primed ahead of
// each generation. Primings are consumed one-shot in FIFO order, so successive
// generations stay isolated from one another.
type PrimedProvider struct {
	queue [][]string
}

// Prime enqueues the exact ordered task list the next generation will return.
func (p *PrimedProvider) Prime(tasks []string) {
	p.queue = enqueueCopy(p.queue, tasks)
}

// enqueueCopy appends a defensive copy of items onto a FIFO priming queue, so a
// caller cannot mutate a queued priming after enqueuing it. All three primed
// providers seed themselves this way.
func enqueueCopy[E any](queue [][]E, items []E) [][]E {
	primed := make([]E, len(items))
	copy(primed, items)
	return append(queue, primed)
}

// dequeue removes and returns the front priming of a FIFO queue, or nil when the
// queue is empty. All three primed providers consume their queues this way.
func dequeue[E any](queue *[][]E) []E {
	if len(*queue) == 0 {
		return nil
	}
	next := (*queue)[0]
	*queue = (*queue)[1:]
	return next
}

// Generate returns the next primed task list, consuming it. The requirement is
// ignored: priming makes output deterministic. When nothing is primed it returns
// an empty list.
func (p *PrimedProvider) Generate(req Requirement) ([]string, error) {
	if next := dequeue(&p.queue); next != nil {
		return next, nil
	}
	return []string{}, nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T23:58:50+05:30","module_hash":"5554c232079497fd8f1f1d9adce3f029975899608be0558460667b130f8dffdf","functions":[{"id":"func/PrimedRiskProvider.PrimeRisks","name":"PrimedRiskProvider.PrimeRisks","line":36,"end_line":38,"hash":"28aaf0c6af73a0017951ab278165796d3bf45e4969407ae7d35303de96e90490"},{"id":"func/PrimedRiskProvider.FlagRisks","name":"PrimedRiskProvider.FlagRisks","line":43,"end_line":45,"hash":"b18b0e3b006e79babdb7c4f928883d31009cf1aa02768bb29a8eaa7d75df581a"},{"id":"func/PrimedEstimateProvider.PrimeEstimates","name":"PrimedEstimateProvider.PrimeEstimates","line":68,"end_line":70,"hash":"ebfbb93d330706dc04fa86b431b806ec28d3bf66a6a5cce993ce4eed57d487db"},{"id":"func/PrimedEstimateProvider.Estimate","name":"PrimedEstimateProvider.Estimate","line":75,"end_line":77,"hash":"6d971b5b47931ae2534c14fc301161d349be3613bb88916347e635ae43586604"},{"id":"func/PrimedProvider.Prime","name":"PrimedProvider.Prime","line":87,"end_line":89,"hash":"e387f277877943c9e63eccff98a094e00b9c509737c133a5ec923359883c8455"},{"id":"func/enqueueCopy","name":"enqueueCopy","line":94,"end_line":98,"hash":"8962dd5319d7fa852ade60ef74dbbec0988f2e16642d2c164b9a034356707f9e"},{"id":"func/dequeue","name":"dequeue","line":102,"end_line":109,"hash":"98418a01ee13b87159dc8bb9c94578ada1018fd51c6a0d4e86857585abab51a4"},{"id":"func/PrimedProvider.Generate","name":"PrimedProvider.Generate","line":114,"end_line":119,"hash":"2c309e9435a4201439b00f06502e7e5e30540fe7dc3bf7795807cf6cb29574a7"}]}
// mutate4go-manifest-end
