package selector

import "errors"

var ErrNoHealthyKeys = errors.New("no healthy keys available")

type Candidate struct {
	KeyID       int64
	CoolingDown bool
	Inflight    int
}

type Selector struct{}

func New() *Selector {
	return &Selector{}
}

func (s *Selector) Select(candidates []Candidate) (Candidate, error) {
	var (
		best  Candidate
		found bool
	)
	for _, candidate := range candidates {
		if candidate.CoolingDown {
			continue
		}
		if !found {
			best = candidate
			found = true
			continue
		}
		if candidate.Inflight < best.Inflight {
			best = candidate
			continue
		}
		if candidate.Inflight == best.Inflight && candidate.KeyID < best.KeyID {
			best = candidate
		}
	}
	if !found {
		return Candidate{}, ErrNoHealthyKeys
	}
	return best, nil
}
