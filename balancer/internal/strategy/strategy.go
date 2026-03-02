package strategy

import "balancer/internal/discovery"

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
	return backends[requests%len(backends)]
}
