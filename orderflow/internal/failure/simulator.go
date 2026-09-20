package failure

import (
	"sync"
	"time"
)

type Scenario string

const (
	Normal          Scenario = "normal"
	Slow            Scenario = "slow"
	PaymentFailure  Scenario = "payment_failure"
	DBFailure       Scenario = "db_failure"
	RedisFailure    Scenario = "redis_failure"
	ErrorSpike      Scenario = "error_spike"
)

type Simulator struct {
	mu sync.RWMutex

	scenario Scenario
}

func NewSimulator() *Simulator {
	return &Simulator{
		scenario: Normal,
	}
}

func (s *Simulator) SetScenario(scenario Scenario) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.scenario = scenario
}

func (s *Simulator) GetScenario() Scenario {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.scenario
}

func (s *Simulator) IsEnabled(scenario Scenario) bool {
	return s.GetScenario() == scenario
}

func (s *Simulator) ApplyLatency() {
	if !s.IsEnabled(Slow) {
		return
	}

	time.Sleep(2 * time.Second)
}