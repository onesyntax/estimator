package webui

import "estimation/internal/wbs"

// EstimateView is a task's 3-point estimate as shown in the Stage-1 estimates
// section.
type EstimateView struct {
	Optimistic  int
	MostLikely  int
	Pessimistic int
	Reasoning   string
}

// TaskRow is one task line in the Stage-1 WBS/estimates list. Metrics holds the
// per-task PERT rollup (expected/SD/RSD) once the task has an estimate; it is
// nil before then. RiskCount is how many risk notes the task carries.
type TaskRow struct {
	Number      int
	Description string
	RiskNotes   []string
	RiskCount   int
	Estimate    *EstimateView
	Metrics     *MetricsView
}

// MetricsView is the project PERT rollup shown in the Stage-1 metrics panel.
type MetricsView struct {
	Expected                  int
	StandardDeviation         int
	RelativeStandardDeviation int
}

// PricingView is the derived pricing strategy shown in the Stage-1 pricing
// panel. Basis is the recommended basis rendered as text, "none" when the band
// refuses a fixed basis (red).
type PricingView struct {
	Flag      string
	RiskLevel string
	Contract  string
	Basis     string
}

// BuildView is the observable state of the Stage-1 Build workspace: the inline
// error, the WBS/estimates list, the two section gates, and the metrics/pricing
// panels that appear once estimates exist. ProposalReachable is the Stage-2
// gate.
type BuildView struct {
	Error              string
	HasWBS             bool
	Tasks              []TaskRow
	RisksLocked        bool
	EstimatesLocked    bool
	EstimatesGenerated bool
	Metrics            *MetricsView
	Pricing            *PricingView
	ProposalReachable  bool
}

// BuildView computes the current Stage-1 screen state from the active WBS.
func (s *Session) BuildView() BuildView {
	v := BuildView{Error: s.err, RisksLocked: true, EstimatesLocked: true}
	cur, ok := s.current()
	if !ok {
		return v
	}
	v.HasWBS = true
	v.Tasks = taskRows(cur.Tasks())
	if cur.Approved() {
		v.RisksLocked = false
		v.EstimatesLocked = false
	}
	if pm, ok := cur.ProjectMetrics(); ok {
		v.EstimatesGenerated = true
		v.Metrics = &MetricsView{
			Expected:                  pm.Expected,
			StandardDeviation:         pm.StandardDeviation,
			RelativeStandardDeviation: pm.RelativeStandardDeviation,
		}
	}
	if ps, ok := cur.PricingStrategy(); ok {
		v.Pricing = &PricingView{
			Flag:      ps.Flag,
			RiskLevel: ps.RiskLevel,
			Contract:  ps.Contract,
			Basis:     basisText(ps.RecommendedBasis),
		}
	}
	v.ProposalReachable = cur.EstimatesApproved()
	return v
}

func taskRows(tasks []wbs.Task) []TaskRow {
	rows := make([]TaskRow, len(tasks))
	for i, t := range tasks {
		rows[i] = TaskRow{
			Number:      i + 1,
			Description: t.Description,
			RiskNotes:   riskNoteTexts(t.RiskNotes),
			RiskCount:   len(t.RiskNotes),
			Estimate:    estimateView(t.Estimate),
			Metrics:     taskMetrics(t.Estimate),
		}
	}
	return rows
}

// taskMetrics renders a task's per-task PERT metrics, or nil when it has no
// estimate yet.
func taskMetrics(e *wbs.Estimate) *MetricsView {
	if e == nil {
		return nil
	}
	m := e.Metrics()
	return &MetricsView{
		Expected:                  m.Expected,
		StandardDeviation:         m.StandardDeviation,
		RelativeStandardDeviation: m.RelativeStandardDeviation,
	}
}

func riskNoteTexts(notes []wbs.RiskNote) []string {
	if len(notes) == 0 {
		return nil
	}
	out := make([]string, len(notes))
	for i, n := range notes {
		out[i] = n.Description
	}
	return out
}

func estimateView(e *wbs.Estimate) *EstimateView {
	if e == nil {
		return nil
	}
	return &EstimateView{
		Optimistic:  e.Optimistic,
		MostLikely:  e.MostLikely,
		Pessimistic: e.Pessimistic,
		Reasoning:   e.Reasoning,
	}
}

// mutate4go-manifest-begin
// {"version":1,"tested_at":"2026-07-09T12:56:51+05:30","module_hash":"8d67210523a15c1f14407b70342ee61373021e438f3cd02e26ac2ed6e9d3b8b5","functions":[{"id":"func/Session.BuildView","name":"Session.BuildView","line":56,"end_line":86,"hash":"0407d2101525459abb1ccb4eab0edcbba5c46d16178f5ca285f57e59a5867460"},{"id":"func/taskRows","name":"taskRows","line":88,"end_line":99,"hash":"dd8c08e6bd1eae87984378ad55ee101d90c9e934ae5c74f0f0aaa09be4dcb5aa"},{"id":"func/riskNoteTexts","name":"riskNoteTexts","line":101,"end_line":110,"hash":"423dcf7f407e997b694ae07e2586216e3599384d0d0c8639fb7fba46f4ac4dc8"},{"id":"func/estimateView","name":"estimateView","line":112,"end_line":122,"hash":"9ba1a6b4b25fc90a3d5f7466ff71a965996ec9c6e0c1ffa5d3f0c0aef05fa824"}]}
// mutate4go-manifest-end
