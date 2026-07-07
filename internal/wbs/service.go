package wbs

import "fmt"

// Service is the estimation service: it generates WBSs from requirement
// documents using a deterministic AI provider and stores them for later edits
// and approval.
type Service struct {
	provider *PrimedProvider
	store    map[string]*WBS
	nextID   int
}

// NewService creates a service backed by a deterministic primed provider and an
// in-memory store.
func NewService() *Service {
	return &Service{
		provider: &PrimedProvider{},
		store:    make(map[string]*WBS),
		nextID:   1,
	}
}

// Prime sets the exact ordered task list the next generation will produce.
func (s *Service) Prime(tasks []string) {
	s.provider.Prime(tasks)
}

// Generate reads the requirement document and, when it is valid, creates and
// stores a new unapproved WBS, returning its id. An empty or unreadable
// document is rejected and no WBS is created.
func (s *Service) Generate(doc RequirementDocument) (string, error) {
	requirement, err := doc.Read()
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

// withWBS looks up the identified WBS and applies action to it, returning
// ErrWBSNotFound when the id is unknown.
func (s *Service) withWBS(id string, action func(*WBS) error) error {
	w, err := s.Get(id)
	if err != nil {
		return err
	}
	return action(w)
}
