package wbs

import (
	"reflect"
	"testing"
)

func TestPrimedProviderReturnsPrimedTasks(t *testing.T) {
	p := &PrimedProvider{}
	p.Prime([]string{"Login API", "Login UI"})

	got, err := p.Generate(Requirement{Text: "any requirement"})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"Login API", "Login UI"}) {
		t.Fatalf("Generate = %v, want the primed tasks", got)
	}
}

func TestPrimingIsOneShotFIFO(t *testing.T) {
	p := &PrimedProvider{}
	p.Prime([]string{"First"})
	p.Prime([]string{"Second"})

	first, _ := p.Generate(Requirement{Text: "req"})
	second, _ := p.Generate(Requirement{Text: "req"})
	if !reflect.DeepEqual(first, []string{"First"}) || !reflect.DeepEqual(second, []string{"Second"}) {
		t.Fatalf("FIFO consumption wrong: first=%v second=%v", first, second)
	}
}
