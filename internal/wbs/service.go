package wbs

import "fmt"

// Service is the estimation service: it generates WBSs from requirement
// documents using an AI provider and stores them for later edits and approval.
// It depends only on the Provider port, so any provider can be injected.
type Service struct {
	provider         Provider
	riskProvider     RiskProvider
	estimateProvider EstimateProvider
	store            map[string]*WBS
	nextID           int
}

// NewService creates a service backed by deterministic primed providers and an
// in-memory store.
func NewService() *Service {
	return NewServiceWithProviders(&PrimedProvider{}, &PrimedRiskProvider{}, &PrimedEstimateProvider{})
}

// NewServiceWithProvider creates a service that generates WBSs with the given
// provider and flags risks and estimates with the default deterministic primed
// providers.
func NewServiceWithProvider(provider Provider) *Service {
	return NewServiceWithProviders(provider, &PrimedRiskProvider{}, &PrimedEstimateProvider{})
}

// NewServiceWithProviders creates a service that generates WBSs, flags risks,
// and estimates with the given providers, storing WBSs in memory. It lets
// callers inject real or fake providers.
func NewServiceWithProviders(provider Provider, riskProvider RiskProvider, estimateProvider EstimateProvider) *Service {
	return &Service{
		provider:         provider,
		riskProvider:     riskProvider,
		estimateProvider: estimateProvider,
		store:            make(map[string]*WBS),
		nextID:           1,
	}
}

// Prime seeds the exact ordered task list the next generation will produce when
// the underlying provider supports priming; it is a no-op otherwise.
func (s *Service) Prime(tasks []string) {
	primeIfSupported(s.provider, tasks, Primer.Prime)
}

// primeIfSupported forwards arg to the provider's priming method when the
// provider implements primer interface I, and does nothing otherwise. It lets
// the service seed a deterministic provider without depending on a concrete type.
func primeIfSupported[I any, A any](provider any, arg A, prime func(I, A)) {
	if p, ok := provider.(I); ok {
		prime(p, arg)
	}
}

// Generate reads the requirement document and, when it is valid, creates and
// stores a new unapproved WBS, returning its id. An empty or unreadable
// document is rejected and no WBS is created.
func (s *Service) Generate(doc RequirementDocument) (string, error) {
	requirement, err := doc.Requirement()
	if err != nil {
		return "", err
	}
	tasks, err := s.provider.Generate(requirement)
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("wbs-%d", s.nextID)
	s.nextID++
	s.store[id] = NewWBS(id, tasks)
	return id, nil
}

// Get returns the stored WBS, or ErrWBSNotFound when the id is unknown.
func (s *Service) Get(id string) (*WBS, error) {
	w, ok := s.store[id]
	if !ok {
		return nil, ErrWBSNotFound
	}
	return w, nil
}

// Count returns the number of stored WBSs.
func (s *Service) Count() int { return len(s.store) }

// AddTask adds a task to the identified WBS.
func (s *Service) AddTask(id, description string) error {
	return s.withWBS(id, func(w *WBS) error { return w.AddTask(description) })
}

// EditTask edits a task in the identified WBS by its one-based number.
func (s *Service) EditTask(id string, number int, description string) error {
	return s.withWBS(id, func(w *WBS) error { return w.EditTask(number, description) })
}

// DeleteTask deletes a task from the identified WBS by its one-based number.
func (s *Service) DeleteTask(id string, number int) error {
	return s.withWBS(id, func(w *WBS) error { return w.DeleteTask(number) })
}

// Approve approves the identified WBS.
func (s *Service) Approve(id string) error {
	return s.withWBS(id, func(w *WBS) error { return w.Approve() })
}

// PrimeRisks seeds the exact risk assignments the next flag will produce when
// the underlying risk provider supports priming; it is a no-op otherwise.
func (s *Service) PrimeRisks(assignments []RiskAssignment) {
	primeIfSupported(s.riskProvider, assignments, RiskPrimer.PrimeRisks)
}

// FlagRisks flags the risks of the identified WBS. The WBS must be approved
// (ErrWBSNotApproved otherwise); flagging replaces every task's existing risk
// notes with the provider's output and leaves the approval state unchanged.
func (s *Service) FlagRisks(id string) error {
	return s.withWBS(id, func(w *WBS) error {
		if !w.Approved() {
			return ErrWBSNotApproved
		}
		assignments, err := s.riskProvider.FlagRisks(w.Tasks())
		if err != nil {
			return err
		}
		w.ReplaceRiskNotes(assignments)
		return nil
	})
}

// AddRiskNote adds a risk note to a task in the identified WBS.
func (s *Service) AddRiskNote(id string, taskNumber int, description string) error {
	return s.withWBS(id, func(w *WBS) error { return w.AddRiskNote(taskNumber, description) })
}

// EditRiskNote edits a risk note on a task in the identified WBS by its
// one-based position.
func (s *Service) EditRiskNote(id string, taskNumber, notePosition int, description string) error {
	return s.withWBS(id, func(w *WBS) error { return w.EditRiskNote(taskNumber, notePosition, description) })
}

// DeleteRiskNote deletes a risk note from a task in the identified WBS by its
// one-based position.
func (s *Service) DeleteRiskNote(id string, taskNumber, notePosition int) error {
	return s.withWBS(id, func(w *WBS) error { return w.DeleteRiskNote(taskNumber, notePosition) })
}

// PrimeEstimates seeds the exact estimate assignments the next generation will
// produce when the underlying estimate provider supports priming; it is a no-op
// otherwise.
func (s *Service) PrimeEstimates(assignments []EstimateAssignment) {
	primeIfSupported(s.estimateProvider, assignments, EstimatePrimer.PrimeEstimates)
}

// GenerateEstimates generates 3-point estimates for the identified WBS. The WBS
// must be approved (ErrWBSNotApprovedForEstimation otherwise); generation
// replaces every task's existing estimate with the provider's output and leaves
// the set unapproved.
func (s *Service) GenerateEstimates(id string) error {
	return s.withWBS(id, func(w *WBS) error {
		if !w.Approved() {
			return ErrWBSNotApprovedForEstimation
		}
		assignments, err := s.estimateProvider.Estimate(w.Tasks())
		if err != nil {
			return err
		}
		w.SetEstimates(assignments)
		return nil
	})
}

// OverrideEstimate overrides a task's estimate in the identified WBS.
func (s *Service) OverrideEstimate(id string, taskNumber int, estimate Estimate) error {
	return s.withWBS(id, func(w *WBS) error { return w.OverrideEstimate(taskNumber, estimate) })
}

// ApproveEstimates approves the estimate set of the identified WBS.
func (s *Service) ApproveEstimates(id string) error {
	return s.withWBS(id, func(w *WBS) error { return w.ApproveEstimates() })
}

// withWBS looks up the identified WBS and applies action to it, returning
// ErrWBSNotFound when the id is unknown.
func (s *Service) withWBS(id string, action func(*WBS) error) error {
	w, err := s.Get(id)
	if err != nil {
		return err
	}
	return action(w)
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T12:31:14+05:30","module_hash":"4f886fc29bce4419a903de3c4a2d1977c385de5a2fb6ed20ca49b425996b14b5","functions":[{"id":"func/NewService","name":"NewService","line":17,"end_line":19,"hash":"5f4d96dd8b1a5efbb90d6018dc4e0b6b6a0d76e4a2d89426714d7aa73b3a93e5"},{"id":"func/NewServiceWithProvider","name":"NewServiceWithProvider","line":23,"end_line":25,"hash":"f03180a168a3507c67d84f62d43099663bec880732209c63715915a20063e0df"},{"id":"func/NewServiceWithProviders","name":"NewServiceWithProviders","line":30,"end_line":37,"hash":"158223be34f59f39b03786a1ebb245b044b3d3464e0aeba5e437b777bda82cd5"},{"id":"func/Service.Prime","name":"Service.Prime","line":41,"end_line":43,"hash":"8931992db3438d1978ca68a6b534cd94e9bd76caa896ad252d2e66c11c8adc5e"},{"id":"func/primeIfSupported","name":"primeIfSupported","line":48,"end_line":52,"hash":"19d0a2228f8d74b4f60821e494175d5991e2c195118c128a8006aaed5d208ac3"},{"id":"func/Service.Generate","name":"Service.Generate","line":57,"end_line":70,"hash":"6298e8a6723a76d8c2d657847cf4c870bdfcae1e2af26e159610d66d74c4bfbd"},{"id":"func/Service.Get","name":"Service.Get","line":73,"end_line":79,"hash":"ba980c14d92f6615a278e2d5125da67991b32f9db88ff5c41103e883a14d8d71"},{"id":"func/Service.Count","name":"Service.Count","line":82,"end_line":82,"hash":"e27c7a2240e0ec3cdd29c0cc241771e70f783eb1754de4f4a42e7c3b173938d6"},{"id":"func/Service.AddTask","name":"Service.AddTask","line":85,"end_line":87,"hash":"448085e030f90d92e6a74fc6ed5d2e2788e73af43aac2b7f634cf199db6584ab"},{"id":"func/Service.EditTask","name":"Service.EditTask","line":90,"end_line":92,"hash":"ec57d85b29e91f98057fe241ccfa212e20bd43bfed153055817979999365024f"},{"id":"func/Service.DeleteTask","name":"Service.DeleteTask","line":95,"end_line":97,"hash":"4ded447389441ec8db14bb3b51ab9719eb235d576496bcc0e12f8cc04a9b4158"},{"id":"func/Service.Approve","name":"Service.Approve","line":100,"end_line":102,"hash":"05d2b3b803e0b0b6132358935235573b0dba93bc204dbc9e42721ef73c8e1fff"},{"id":"func/Service.PrimeRisks","name":"Service.PrimeRisks","line":106,"end_line":108,"hash":"f21925a62ae4f6e1049f578fc288ee25d31a90f3ef1cda03740f97d5f6bc89f5"},{"id":"func/Service.FlagRisks","name":"Service.FlagRisks","line":113,"end_line":125,"hash":"3818e19451a424759345dba5d9b1827dc7fe90f830cab65368bf7dab2242010e"},{"id":"func/Service.AddRiskNote","name":"Service.AddRiskNote","line":128,"end_line":130,"hash":"0edf0430a7c795930d27eca29e388078f4da32124714140dec8ee5bd2197230c"},{"id":"func/Service.EditRiskNote","name":"Service.EditRiskNote","line":134,"end_line":136,"hash":"3d8f80bc484153d1eddfc95523b81a3b56dec20f51cbf08f233f241b83263b73"},{"id":"func/Service.DeleteRiskNote","name":"Service.DeleteRiskNote","line":140,"end_line":142,"hash":"dba91c5a9c8709108aafdeb616e4623e9927f9e739bb7e0bc9c0e9537334570c"},{"id":"func/Service.withWBS","name":"Service.withWBS","line":146,"end_line":152,"hash":"65e497123cd1141ff731f54459a5e5ff14675133710d871c2030f395082b8bfb"}]}
// mutate4go-manifest-end
