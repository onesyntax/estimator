package wbs

import "fmt"

// Service is the estimation service: it generates WBSs from requirement
// documents using an AI provider and stores them for later edits and approval.
// It depends only on the Provider port, so any provider can be injected.
type Service struct {
	provider     Provider
	riskProvider RiskProvider
	store        map[string]*WBS
	nextID       int
}

// NewService creates a service backed by deterministic primed providers and an
// in-memory store.
func NewService() *Service {
	return NewServiceWithProviders(&PrimedProvider{}, &PrimedRiskProvider{})
}

// NewServiceWithProvider creates a service that generates WBSs with the given
// provider and flags risks with the default deterministic primed risk provider.
func NewServiceWithProvider(provider Provider) *Service {
	return NewServiceWithProviders(provider, &PrimedRiskProvider{})
}

// NewServiceWithProviders creates a service that generates WBSs with the given
// generation provider and flags risks with the given risk provider, storing
// WBSs in memory. It lets callers inject real or fake providers.
func NewServiceWithProviders(provider Provider, riskProvider RiskProvider) *Service {
	return &Service{
		provider:     provider,
		riskProvider: riskProvider,
		store:        make(map[string]*WBS),
		nextID:       1,
	}
}

// Prime seeds the exact ordered task list the next generation will produce when
// the underlying provider supports priming; it is a no-op otherwise.
func (s *Service) Prime(tasks []string) {
	if p, ok := s.provider.(Primer); ok {
		p.Prime(tasks)
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
	if p, ok := s.riskProvider.(RiskPrimer); ok {
		p.PrimeRisks(assignments)
	}
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
// {"version":1,"tested_at":"2026-07-07T22:19:04+05:30","module_hash":"22df83703763397efe6d1b16ee137a0cf982a62f495b5b1186714b90431db77a","functions":[{"id":"func/NewService","name":"NewService","line":16,"end_line":18,"hash":"490a64047c3f350cd03213e1486ceeb86546d6d675a9a674a991c16e0282af1b"},{"id":"func/NewServiceWithProvider","name":"NewServiceWithProvider","line":23,"end_line":29,"hash":"d9edda2945942eed61500b31f4e937708a7e0d6213bca7eb5591d2b82ed74018"},{"id":"func/Service.Prime","name":"Service.Prime","line":33,"end_line":37,"hash":"c131a553d735024a98100fdbf300908120c918f4dd0082210ff47ef182cf2bff"},{"id":"func/Service.Generate","name":"Service.Generate","line":42,"end_line":55,"hash":"730843b3078a9c757e082e7c7e2681a2f8916a29727f2a409ededc8f3501e804"},{"id":"func/Service.Get","name":"Service.Get","line":58,"end_line":64,"hash":"ba980c14d92f6615a278e2d5125da67991b32f9db88ff5c41103e883a14d8d71"},{"id":"func/Service.Count","name":"Service.Count","line":67,"end_line":67,"hash":"e27c7a2240e0ec3cdd29c0cc241771e70f783eb1754de4f4a42e7c3b173938d6"},{"id":"func/Service.AddTask","name":"Service.AddTask","line":70,"end_line":72,"hash":"448085e030f90d92e6a74fc6ed5d2e2788e73af43aac2b7f634cf199db6584ab"},{"id":"func/Service.EditTask","name":"Service.EditTask","line":75,"end_line":77,"hash":"ec57d85b29e91f98057fe241ccfa212e20bd43bfed153055817979999365024f"},{"id":"func/Service.DeleteTask","name":"Service.DeleteTask","line":80,"end_line":82,"hash":"4ded447389441ec8db14bb3b51ab9719eb235d576496bcc0e12f8cc04a9b4158"},{"id":"func/Service.Approve","name":"Service.Approve","line":85,"end_line":87,"hash":"05d2b3b803e0b0b6132358935235573b0dba93bc204dbc9e42721ef73c8e1fff"},{"id":"func/Service.withWBS","name":"Service.withWBS","line":91,"end_line":97,"hash":"65e497123cd1141ff731f54459a5e5ff14675133710d871c2030f395082b8bfb"}]}
// mutate4go-manifest-end
