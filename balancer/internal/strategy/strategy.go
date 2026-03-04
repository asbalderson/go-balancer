package strategy

import (
	"balancer/internal/discovery"

	"pkg/logging"
)

type Strategy interface {
	Next(backends []discovery.Backend, requests int) discovery.Backend
}

func NewStrategy(method string) Strategy {
	switch method {
	case "RoundRobin":
		return &RoundRobin{}
	default:
		return &RoundRobin{}
	}
}

type RoundRobin struct{}

func (rr RoundRobin) Next(backends []discovery.Backend, requests int) discovery.Backend {
	next := backends[requests%len(backends)]
	logging.Debug("Backend requested, sending %s", next)
	return next
}
