package wbs

import (
	"fmt"
	"strings"
)

// Task is a single unit of work in a WBS. Its ID is stable for the lifetime of
// the task even as tasks around it are added or deleted.
type Task struct {
	ID          string
	Description string
}

// WBS (Work Breakdown Structure) is the aggregate produced from a requirement
// document. Tasks are ordered and referred to by their one-based position
// ("task number N"). Any structural edit unapproves the WBS.
type WBS struct {
	id       string
	tasks    []Task
	approved bool
	nextSeq  int
}

// NewWBS builds an unapproved WBS from the given ordered task descriptions.
func NewWBS(id string, descriptions []string) *WBS {
	w := &WBS{id: id, nextSeq: 1}
	for _, d := range descriptions {
		w.tasks = append(w.tasks, Task{ID: w.newTaskID(), Description: d})
	}
	return w
}

func (w *WBS) newTaskID() string {
	id := fmt.Sprintf("t%d", w.nextSeq)
	w.nextSeq++
	return id
}

// ID returns the WBS identifier.
func (w *WBS) ID() string { return w.id }

// Approved reports whether the WBS is currently approved.
func (w *WBS) Approved() bool { return w.approved }

// TaskCount returns the number of tasks in the WBS.
func (w *WBS) TaskCount() int { return len(w.tasks) }

// Tasks returns a copy of the ordered task list.
func (w *WBS) Tasks() []Task {
	out := make([]Task, len(w.tasks))
	copy(out, w.tasks)
	return out
}

// TaskAt returns the task at the given one-based number, or ok=false when the
// number is out of range.
func (w *WBS) TaskAt(number int) (Task, bool) {
	if number < 1 || number > len(w.tasks) {
		return Task{}, false
	}
	return w.tasks[number-1], true
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
// {"version":1,"tested_at":"2026-07-07T22:19:03+05:30","module_hash":"b9a1f1ee48d880a1c4704f06e9ff600bca9873017a7d9e12b8e1504e1d2d11a6","functions":[{"id":"func/NewWBS","name":"NewWBS","line":26,"end_line":32,"hash":"e8427c631f4dd6053e4056d4a7e78334f008e2413c295abe58946f3cb1f3c7c1"},{"id":"func/WBS.newTaskID","name":"WBS.newTaskID","line":34,"end_line":38,"hash":"0aeb9ce02ef04c8189cded047be8d6ec19012af82d045fcb598059143a7687de"},{"id":"func/WBS.ID","name":"WBS.ID","line":41,"end_line":41,"hash":"8c1f164fcec1326c50aeced7037fbff44abb02f34bfb5fe95a5a806a3e2327c5"},{"id":"func/WBS.Approved","name":"WBS.Approved","line":44,"end_line":44,"hash":"e2faf61713da11f50aa39a0b690c90f60d3a917c7ce7060dfebbfeb827abcd12"},{"id":"func/WBS.TaskCount","name":"WBS.TaskCount","line":47,"end_line":47,"hash":"487c3f0cac21d27fcd443f7bb1d411a49a0d9c82a801b4e4b1116cd68c8fd34c"},{"id":"func/WBS.Tasks","name":"WBS.Tasks","line":50,"end_line":54,"hash":"9439a189c9de4edc3049caf19700339ddd0338029996da08a7d9a5b838b26832"},{"id":"func/WBS.TaskAt","name":"WBS.TaskAt","line":58,"end_line":63,"hash":"a778228ff355f96cedbd82d30a98eed84de6525fbaeb47b93d53cf585b49291e"},{"id":"func/WBS.ApprovedTasks","name":"WBS.ApprovedTasks","line":67,"end_line":72,"hash":"720afdca37b56230fd03f41baed7c4bc49ec0e1944dad6beacb10a863c9c2db2"},{"id":"func/WBS.AddTask","name":"WBS.AddTask","line":75,"end_line":83,"hash":"bb13c2bc3cf332ddce83c653166f1badcf374de9a7b72142e9164f0397dab834"},{"id":"func/WBS.EditTask","name":"WBS.EditTask","line":87,"end_line":98,"hash":"29e07a7c40c3c1514f735f4acdad032760a07dcde3abb5e7e03ea13dac3a3aa6"},{"id":"func/WBS.DeleteTask","name":"WBS.DeleteTask","line":101,"end_line":108,"hash":"d9b11380ef95a228ce7df8d14a1477703518a4f644e219170e6c2f0345d9d842"},{"id":"func/WBS.Approve","name":"WBS.Approve","line":111,"end_line":117,"hash":"7f578a0905ec5c5aaef6d700860c5152ae380e54e320041d8de25d0cf49389ed"},{"id":"func/requireDescription","name":"requireDescription","line":119,"end_line":125,"hash":"435a1b1682780b585ce996a4eee6bbf9dbdd5a72a3796037ce5bfe3e36156753"}]}
// mutate4go-manifest-end
