package failure

import (
	"testing"
)

func TestSimulatorScenarios(t *testing.T) {
	sim := NewSimulator()

	if sim.GetScenario() != Normal {
		t.Fatalf("expected initial scenario to be %s, got %s", Normal, sim.GetScenario())
	}

	scenarios := []Scenario{
		Slow,
		PaymentFailure,
		DBFailure,
		RedisFailure,
		ErrorSpike,
		Normal,
	}

	for _, sc := range scenarios {
		sim.SetScenario(sc)
		if sim.GetScenario() != sc {
			t.Errorf("expected scenario %s, got %s", sc, sim.GetScenario())
		}
		if !sim.IsEnabled(sc) {
			t.Errorf("expected IsEnabled(%s) to be true", sc)
		}
		for _, other := range scenarios {
			if other != sc && sim.IsEnabled(other) {
				t.Errorf("expected IsEnabled(%s) to be false when active is %s", other, sc)
			}
		}
	}
}
