package selector

import "testing"

func TestSelectorSkipsCoolingDownKeys(t *testing.T) {
	s := New()
	chosen, err := s.Select([]Candidate{
		{KeyID: 1, CoolingDown: true, Inflight: 0},
		{KeyID: 2, CoolingDown: false, Inflight: 1},
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if chosen.KeyID != 2 {
		t.Fatalf("expected key 2, got %d", chosen.KeyID)
	}
}

func TestSelectorPrefersLeastInflight(t *testing.T) {
	s := New()
	chosen, err := s.Select([]Candidate{
		{KeyID: 7, Inflight: 4},
		{KeyID: 9, Inflight: 1},
		{KeyID: 3, Inflight: 2},
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if chosen.KeyID != 9 {
		t.Fatalf("expected key 9, got %d", chosen.KeyID)
	}
}

func TestSelectorTieBreaksByKeyID(t *testing.T) {
	s := New()
	chosen, err := s.Select([]Candidate{
		{KeyID: 5, Inflight: 2},
		{KeyID: 2, Inflight: 2},
	})
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if chosen.KeyID != 2 {
		t.Fatalf("expected key 2, got %d", chosen.KeyID)
	}
}

func TestSelectorErrorsWhenNoEligibleKeys(t *testing.T) {
	s := New()
	if _, err := s.Select([]Candidate{{KeyID: 1, CoolingDown: true}}); err == nil {
		t.Fatalf("expected error")
	}
}
