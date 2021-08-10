// Copyright 2021 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package caasmodelconfigmanager

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names/v4"
	"github.com/juju/worker/v2"
	"github.com/juju/worker/v2/catacomb"

	"github.com/juju/juju/api/base"
	caasmodelconfigmanagerapi "github.com/juju/juju/api/caasmodelconfigmanager"
	"github.com/juju/juju/controller"
	"github.com/juju/juju/core/watcher"
	"github.com/juju/juju/docker"
)

// Logger represents the methods used by the worker to log details.
type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Errorf(string, ...interface{})
	Warningf(string, ...interface{})
	Tracef(string, ...interface{})

	Child(string) loggo.Logger
}

type Facade interface {
	// TODO: move to facade!
	ControllerConfig() (controller.Config, error)
	WatchControllerConfig() (watcher.NotifyWatcher, error)
}

type CAASBroker interface {
	EnsureImageRepoSecret(docker.ImageRepoDetails) error
}

// Config holds the configuration and dependencies for a worker.
type Config struct {
	ModelTag names.ModelTag

	Facade Facade
	Broker CAASBroker
}

// Validate returns an error if the config cannot be expected
// to drive a functional worker.
func (config Config) Validate() error {
	if config.Facade == nil {
		return errors.NotValidf("nil Facade")
	}
	if config.Broker == nil {
		return errors.NotValidf("nil Broker")
	}

	if config.ModelTag == (names.ModelTag{}) {
		return errors.NotValidf("empty ModelTag")
	}
	return nil
}

type manager struct {
	catacomb catacomb.Catacomb

	name          string
	config        Config
	imageRepoInfo docker.ImageRepoDetails
}

func NewFacade(caller base.APICaller) Facade {
	return caasmodelconfigmanagerapi.NewClient(caller)
}

// NewWorker returns a worker that unlocks the model upgrade gate.
func NewWorker(config Config) (worker.Worker, error) {
	if err := config.Validate(); err != nil {
		return nil, errors.Trace(err)
	}
	w := &manager{
		name:   config.ModelTag.Id(),
		config: config,
	}
	err := catacomb.Invoke(catacomb.Plan{
		Site: &w.catacomb,
		Work: w.loop,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return w, nil
}

// Kill is part of the worker.Worker interface.
func (w *manager) Kill() {
	w.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *manager) Wait() error {
	return w.catacomb.Wait()
}

func (w *manager) loop() error {
	controllerConfigWatcher, err := w.config.Facade.WatchControllerConfig()
	if err != nil {
		return errors.Trace(err)
	}
	if err := w.catacomb.Add(controllerConfigWatcher); err != nil {
		return errors.Trace(err)
	}
	for {
		select {
		case <-w.catacomb.Dying():
			return w.catacomb.ErrDying()
		case _, ok := <-controllerConfigWatcher.Changes():
			if !ok {
				return fmt.Errorf("controller config watcher %q closed channel", w.name)
			}
			controllerConfig, err := w.config.Facade.ControllerConfig()
			if err != nil {
				return errors.Trace(err)
			}
			newImageRepoInfo := controllerConfig.CAASImageRepo()
			if newImageRepoInfo != nil && !w.imageRepoInfo.AuthEqual(*newImageRepoInfo) {
				if err := w.config.Broker.EnsureImageRepoSecret(*newImageRepoInfo); err != nil {
					return errors.Trace(err)
				}
				w.imageRepoInfo = *newImageRepoInfo
			}
		}
	}
}
