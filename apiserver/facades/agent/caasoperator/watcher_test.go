// Copyright 2019 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package caasoperator_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/facades/agent/caasoperator"
	"github.com/juju/juju/core/watcher/watchertest"
	"github.com/juju/juju/state"
	"github.com/juju/juju/state/testing"
	coretesting "github.com/juju/juju/testing"
)

var _ = gc.Suite(&IDWatcherSuite{})

type IDWatcherSuite struct {
	coretesting.BaseSuite
}

func (s *IDWatcherSuite) TestWatcher(c *gc.C) {
	m := &mockModel{}
	m.containers = []state.CloudContainer{
		&mockCloudContainer{unit: "A", providerID: "a"},
		&mockCloudContainer{unit: "C", providerID: "c"},
	}
	wc := make(chan []string, 4)
	wc <- []string{"a"}
	wc <- []string{"b"}
	wc <- []string{"c"}
	wc <- []string{"d"}
	srcWatcher := watchertest.NewMockStringsWatcher(wc)
	idWatcher, err := caasoperator.NewUnitIDWatcher(m, srcWatcher)
	c.Assert(err, jc.ErrorIsNil)

	testWatcher := testing.NewStringsWatcherC(c, s, idWatcher)
	testWatcher.AssertChangeInSingleEvent("A")
	testWatcher.AssertChangeInSingleEvent("C")

	err = idWatcher.Stop()
	c.Assert(err, jc.ErrorIsNil)
}

// StartSync fulfills testing.StartSync interface.
func (s *IDWatcherSuite) StartSync() {
}
