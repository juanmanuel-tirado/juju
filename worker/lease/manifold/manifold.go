// Copyright 2018 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package manifold

// This needs to be in a different package from the lease manager
// because it uses state (to construct the raftlease store), but the
// lease manager also runs as a worker in state, so the state package
// depends on worker/lease. Having it in worker/lease produces an
// import cycle.

import (
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/pubsub/v2"
	"github.com/juju/worker/v3"
	"github.com/juju/worker/v3/dependency"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/juju/juju/agent"
	corelease "github.com/juju/juju/core/lease"
	"github.com/juju/juju/core/raftlease"
	"github.com/juju/juju/worker/common"
	"github.com/juju/juju/worker/lease"
)

type Logger interface {
	Errorf(string, ...interface{})
	Warningf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Tracef(string, ...interface{})
}

const (
	// MaxSleep is the longest the manager will sleep before checking
	// whether any leases should be expired. If it can see a lease
	// expiring sooner than that it will still wake up earlier.
	MaxSleep = time.Minute

	// ForwardTimeout is how long the store should wait for a response
	// after sending a lease operation over the hub before deciding a
	// a response is never coming back (for example if we send the
	// request during a raft-leadership election). This should be long
	// enough that we can be very confident the request was missed.
	ForwardTimeout = 5 * time.Second
)

// TODO(raftlease): This manifold does too much - split out a worker
// that holds the lease store and a manifold that creates it. Then
// make this one depend on that.

// ManifoldConfig holds the resources needed to start the lease
// manager in a dependency engine.
type ManifoldConfig struct {
	AgentName      string
	ClockName      string
	CentralHubName string

	FSM                  *raftlease.FSM
	RequestTopic         string
	Logger               lease.Logger
	LogDir               string
	PrometheusRegisterer prometheus.Registerer
	NewWorker            func(lease.ManagerConfig) (worker.Worker, error)
	NewStore             func(raftlease.StoreConfig) *raftlease.Store
	NewClient            ClientFunc
}

// Validate checks that the config has all the required values.
func (c ManifoldConfig) Validate() error {
	if c.AgentName == "" {
		return errors.NotValidf("empty AgentName")
	}
	if c.ClockName == "" {
		return errors.NotValidf("empty ClockName")
	}
	if c.CentralHubName == "" {
		return errors.NotValidf("empty CentralHubName")
	}
	if c.FSM == nil {
		return errors.NotValidf("nil FSM")
	}
	if c.RequestTopic == "" {
		return errors.NotValidf("empty RequestTopic")
	}
	if c.Logger == nil {
		return errors.NotValidf("nil Logger")
	}
	if c.PrometheusRegisterer == nil {
		return errors.NotValidf("nil PrometheusRegisterer")
	}
	if c.NewWorker == nil {
		return errors.NotValidf("nil NewWorker")
	}
	if c.NewStore == nil {
		return errors.NotValidf("nil NewStore")
	}
	if c.NewClient == nil {
		return errors.NotValidf("nil NewClient")
	}
	return nil
}

type manifoldState struct {
	config ManifoldConfig
	store  *raftlease.Store
}

func (s *manifoldState) start(context dependency.Context) (worker.Worker, error) {
	if err := s.config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}

	var agent agent.Agent
	if err := context.Get(s.config.AgentName, &agent); err != nil {
		return nil, errors.Trace(err)
	}

	var clock clock.Clock
	if err := context.Get(s.config.ClockName, &clock); err != nil {
		return nil, errors.Trace(err)
	}

	var hub *pubsub.StructuredHub
	if err := context.Get(s.config.CentralHubName, &hub); err != nil {
		return nil, errors.Trace(err)
	}

	metrics := raftlease.NewOperationClientMetrics(clock)
	client := s.config.NewClient(hub, s.config.RequestTopic, clock, metrics)
	s.store = s.config.NewStore(raftlease.StoreConfig{
		FSM:              s.config.FSM,
		Client:           client,
		Clock:            clock,
		MetricsCollector: metrics,
	})

	controllerUUID := agent.CurrentConfig().Controller().Id()
	w, err := s.config.NewWorker(lease.ManagerConfig{
		Secretary:            lease.SecretaryFinder(controllerUUID),
		Store:                s.store,
		Clock:                clock,
		Logger:               s.config.Logger,
		MaxSleep:             MaxSleep,
		EntityUUID:           controllerUUID,
		LogDir:               s.config.LogDir,
		PrometheusRegisterer: s.config.PrometheusRegisterer,
	})
	return w, errors.Trace(err)
}

func (s *manifoldState) output(in worker.Worker, out interface{}) error {
	if w, ok := in.(*common.CleanupWorker); ok {
		in = w.Worker
	}
	manager, ok := in.(*lease.Manager)
	if !ok {
		return errors.Errorf("expected input of type *worker/lease.Manager, got %T", in)
	}
	switch out := out.(type) {
	case *corelease.Manager:
		*out = manager
		return nil
	default:
		return errors.Errorf("expected output of type *core/lease.Manager, got %T", out)
	}
}

// Manifold builds a dependency.Manifold for running a lease manager.
func Manifold(config ManifoldConfig) dependency.Manifold {
	s := manifoldState{config: config}
	return dependency.Manifold{
		Inputs: []string{
			config.AgentName,
			config.ClockName,
			config.CentralHubName,
		},
		Start:  s.start,
		Output: s.output,
	}
}

// NewWorker wraps NewManager to return worker.Worker for testability.
func NewWorker(config lease.ManagerConfig) (worker.Worker, error) {
	return lease.NewManager(config)
}

// NewStore is a shim to make a raftlease.Store for testability.
func NewStore(config raftlease.StoreConfig) *raftlease.Store {
	return raftlease.NewStore(config)
}

type ClientFunc = func(*pubsub.StructuredHub, string, clock.Clock, *raftlease.OperationClientMetrics) raftlease.Client

// NewClientFunc returns a lease API client based on pub/sub.
func NewClientFunc(
	hub *pubsub.StructuredHub,
	requestTopic string,
	clock clock.Clock,
	metrics *raftlease.OperationClientMetrics,
) raftlease.Client {
	return raftlease.NewPubsubClient(raftlease.PubsubClientConfig{
		Hub:            hub,
		RequestTopic:   requestTopic,
		Clock:          clock,
		ForwardTimeout: ForwardTimeout,
		ClientMetrics:  metrics,
	})
}
