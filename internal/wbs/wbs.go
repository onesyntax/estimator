package wbs

import (
	"fmt"
	"strings"
)

// Task is a single unit of work in a WBS. Its ID is stable for the lifetime of
// the task even as tasks around it are added or deleted. RiskNotes are the
// Tech Lead's risk annotations for the task, ordered and referred to by their
// one-based position ("risk note K").
type Task struct {
	ID          string
	Description string
	RiskNotes   []RiskNote
	Estimate    *Estimate
}

// clone returns a deep copy of the task so callers cannot mutate the WBS's
// internal risk notes or estimate through a returned Task value.
func (t Task) clone() Task {
	if t.RiskNotes != nil {
		notes := make([]RiskNote, len(t.RiskNotes))
		copy(notes, t.RiskNotes)
		t.RiskNotes = notes
	}
	if t.Estimate != nil {
		estimate := *t.Estimate
		t.Estimate = &estimate
	}
	return t
}

// WBS (Work Breakdown Structure) is the aggregate produced from a requirement
// document. Tasks are ordered and referred to by their one-based position
// ("task number N"). Any structural edit unapproves the WBS.
type WBS struct {
	id                 string
	tasks              []Task
	approved           bool
	nextSeq            int
	nextNote           int
	estimatesGenerated bool
	estimatesApproved  bool
}

// NewWBS builds an unapproved WBS from the given ordered task descriptions.
func NewWBS(id string, descriptions []string) *WBS {
	w := &WBS{id: id, nextSeq: 1, nextNote: 1}
	for _, d := range descriptions {
		w.tasks = append(w.tasks, Task{ID: w.newTaskID(), Description: d})
	}
	return w
}

func (w *WBS) newTaskID() string { return nextID("t", &w.nextSeq) }

// nextID formats the next identifier for a monotonic counter as prefix+count,
// then advances the counter. Task ids and risk-note ids share this scheme.
func nextID(prefix string, counter *int) string {
	id := fmt.Sprintf("%s%d", prefix, *counter)
	*counter++
	return id
}

// ID returns the WBS identifier.
func (w *WBS) ID() string { return w.id }

// Approved reports whether the WBS is currently approved.
func (w *WBS) Approved() bool { return w.approved }

// TaskCount returns the number of tasks in the WBS.
func (w *WBS) TaskCount() int { return len(w.tasks) }

// Tasks returns a deep copy of the ordered task list, including each task's
// risk notes, so callers cannot mutate the WBS's internal state.
func (w *WBS) Tasks() []Task {
	out := make([]Task, len(w.tasks))
	for i, t := range w.tasks {
		out[i] = t.clone()
	}
	return out
}

// TaskAt returns a deep copy of the task at the given one-based number, or
// ok=false when the number is out of range.
func (w *WBS) TaskAt(number int) (Task, bool) {
	if number < 1 || number > len(w.tasks) {
		return Task{}, false
	}
	return w.tasks[number-1].clone(), true
}

// ApprovedTasks returns the approved task list (the module output) while the
// WBS is approved, and nil otherwise.
func (w *WBS) ApprovedTasks() []Task {
	if !w.approved {
		return nil
	}
	return w.Tasks()
}

// AddTask appends a task with the given description and unapproves the WBS.
func (w *WBS) AddTask(description string) error {
	description, err := requireDescription(description)
	if err != nil {
		return err
	}
	w.tasks = append(w.tasks, Task{ID: w.newTaskID(), Description: description})
	w.approved = false
	return nil
}

// EditTask replaces the description of the task at the one-based number and
// unapproves the WBS.
func (w *WBS) EditTask(number int, description string) error {
	if number < 1 || number > len(w.tasks) {
		return ErrTaskNotFound
	}
	description, err := requireDescription(description)
	if err != nil {
		return err
	}
	w.tasks[number-1].Description = description
	w.approved = false
	return nil
}

// DeleteTask removes the task at the one-based number and unapproves the WBS.
func (w *WBS) DeleteTask(number int) error {
	if number < 1 || number > len(w.tasks) {
		return ErrTaskNotFound
	}
	w.tasks = append(w.tasks[:number-1], w.tasks[number:]...)
	w.approved = false
	return nil
}

// Approve marks the WBS approved. It fails when the WBS has no tasks.
func (w *WBS) Approve() error {
	if len(w.tasks) == 0 {
		return ErrNoTasks
	}
	w.approved = true
	return nil
}

func requireDescription(description string) (string, error) {
	trimmed := strings.TrimSpace(description)
	if trimmed == "" {
		return "", ErrEmptyDescription
	}
	return trimmed, nil
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-08T15:50:32+05:30","module_hash":"bafb29373906ec31aebc9361b83cb1dc6aa8c6cab86b171fcf1c640212b5df4d","functions":[{"id":"func/Task.clone","name":"Task.clone","line":21,"end_line":32,"hash":"c1b8025727b6e4e71a412a4f773c2b04afc936bfa9881143e3c66c16bcb830f5"},{"id":"func/NewWBS","name":"NewWBS","line":48,"end_line":54,"hash":"8dd937a1bc628ff6020730a44daa35064238fc2e055a23c7bb22fd3376b774bc"},{"id":"func/WBS.newTaskID","name":"WBS.newTaskID","line":56,"end_line":56,"hash":"2b151645de9ae1f9c2042552b55134545003dca352ba17938424502952dc4d6c"},{"id":"func/nextID","name":"nextID","line":60,"end_line":64,"hash":"2eae52c77b07dfbda0935c0f5eb45efa59095a17b79224b2b6508a521a9be572"},{"id":"func/WBS.ID","name":"WBS.ID","line":67,"end_line":67,"hash":"8c1f164fcec1326c50aeced7037fbff44abb02f34bfb5fe95a5a806a3e2327c5"},{"id":"func/WBS.Approved","name":"WBS.Approved","line":70,"end_line":70,"hash":"e2faf61713da11f50aa39a0b690c90f60d3a917c7ce7060dfebbfeb827abcd12"},{"id":"func/WBS.TaskCount","name":"WBS.TaskCount","line":73,"end_line":73,"hash":"487c3f0cac21d27fcd443f7bb1d411a49a0d9c82a801b4e4b1116cd68c8fd34c"},{"id":"func/WBS.Tasks","name":"WBS.Tasks","line":77,"end_line":83,"hash":"a108c9388489565be60d059769de81bc7359dd423c3aac5c2700a1f042106e84"},{"id":"func/WBS.TaskAt","name":"WBS.TaskAt","line":87,"end_line":92,"hash":"09fa08dd6ebdbaead37c57fb1a6ed213623a5f6c5f2e253c4d27486940a7160d"},{"id":"func/WBS.ApprovedTasks","name":"WBS.ApprovedTasks","line":96,"end_line":101,"hash":"720afdca37b56230fd03f41baed7c4bc49ec0e1944dad6beacb10a863c9c2db2"},{"id":"func/WBS.AddTask","name":"WBS.AddTask","line":104,"end_line":112,"hash":"bb13c2bc3cf332ddce83c653166f1badcf374de9a7b72142e9164f0397dab834"},{"id":"func/WBS.EditTask","name":"WBS.EditTask","line":116,"end_line":127,"hash":"29e07a7c40c3c1514f735f4acdad032760a07dcde3abb5e7e03ea13dac3a3aa6"},{"id":"func/WBS.DeleteTask","name":"WBS.DeleteTask","line":130,"end_line":137,"hash":"d9b11380ef95a228ce7df8d14a1477703518a4f644e219170e6c2f0345d9d842"},{"id":"func/WBS.Approve","name":"WBS.Approve","line":140,"end_line":146,"hash":"7f578a0905ec5c5aaef6d700860c5152ae380e54e320041d8de25d0cf49389ed"},{"id":"func/requireDescription","name":"requireDescription","line":148,"end_line":154,"hash":"435a1b1682780b585ce996a4eee6bbf9dbdd5a72a3796037ce5bfe3e36156753"}]}
// mutate4go-manifest-end
