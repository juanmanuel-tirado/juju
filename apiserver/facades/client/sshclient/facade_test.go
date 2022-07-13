// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package sshclient_test

import (
	"github.com/golang/mock/gomock"
	"github.com/juju/errors"
	"github.com/juju/names/v4"
	jujutesting "github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facades/client/sshclient"
	apiservertesting "github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/core/network"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/context"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/state"
	"github.com/juju/juju/testing"
)

type facadeSuite struct {
	testing.BaseSuite
	backend          *mockBackend
	authorizer       *apiservertesting.FakeAuthorizer
	facade           *sshclient.Facade
	m0, uFoo, uOther string

	callContext context.ProviderCallContext
}

var _ = gc.Suite(&facadeSuite{})

func (s *facadeSuite) SetUpSuite(c *gc.C) {
	s.BaseSuite.SetUpSuite(c)
	s.m0 = names.NewMachineTag("0").String()
	s.uFoo = names.NewUnitTag("foo/0").String()
	s.uOther = names.NewUnitTag("other/1").String()
}

func (s *facadeSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)

	s.backend = new(mockBackend)
	s.authorizer = new(apiservertesting.FakeAuthorizer)
	s.authorizer.Tag = names.NewUserTag("igor")
	s.authorizer.AdminTag = names.NewUserTag("igor")

	s.callContext = context.NewEmptyCloudCallContext()
	facade, err := sshclient.InternalFacade(s.backend, nil, s.authorizer, s.callContext)
	c.Assert(err, jc.ErrorIsNil)
	s.facade = facade
}

func (s *facadeSuite) TestMachineAuthNotAllowed(c *gc.C) {
	s.authorizer.Tag = names.NewMachineTag("0")
	_, err := sshclient.InternalFacade(s.backend, nil, s.authorizer, s.callContext)
	c.Assert(err, gc.Equals, apiservererrors.ErrPerm)
}

func (s *facadeSuite) TestUnitAuthNotAllowed(c *gc.C) {
	s.authorizer.Tag = names.NewUnitTag("foo/0")
	_, err := sshclient.InternalFacade(s.backend, nil, s.authorizer, s.callContext)
	c.Assert(err, gc.Equals, apiservererrors.ErrPerm)
}

func (s *facadeSuite) TestPublicAddress(c *gc.C) {
	args := params.Entities{
		Entities: []params.Entity{{s.m0}, {s.uFoo}, {s.uOther}},
	}
	results, err := s.facade.PublicAddress(args)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(results, gc.DeepEquals, params.SSHAddressResults{
		Results: []params.SSHAddressResult{
			{Address: "1.1.1.1"},
			{Address: "3.3.3.3"},
			{Error: apiservertesting.NotFoundError("entity")},
		},
	})
	s.backend.stub.CheckCalls(c, []jujutesting.StubCall{
		{"GetMachineForEntity", []interface{}{s.m0}},
		{"GetMachineForEntity", []interface{}{s.uFoo}},
		{"GetMachineForEntity", []interface{}{s.uOther}},
	})
}

func (s *facadeSuite) TestPrivateAddress(c *gc.C) {
	args := params.Entities{
		Entities: []params.Entity{{s.uOther}, {s.m0}, {s.uFoo}},
	}
	results, err := s.facade.PrivateAddress(args)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(results, gc.DeepEquals, params.SSHAddressResults{
		Results: []params.SSHAddressResult{
			{Error: apiservertesting.NotFoundError("entity")},
			{Address: "2.2.2.2"},
			{Address: "4.4.4.4"},
		},
	})
	s.backend.stub.CheckCalls(c, []jujutesting.StubCall{
		{"GetMachineForEntity", []interface{}{s.uOther}},
		{"GetMachineForEntity", []interface{}{s.m0}},
		{"GetMachineForEntity", []interface{}{s.uFoo}},
	})
}

func (s *facadeSuite) TestAllAddresses(c *gc.C) {
	args := params.Entities{
		Entities: []params.Entity{{s.uOther}, {s.m0}, {s.uFoo}},
	}
	results, err := s.facade.AllAddresses(args)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(results, gc.DeepEquals, params.SSHAddressesResults{
		Results: []params.SSHAddressesResult{
			{Error: apiservertesting.NotFoundError("entity")},
			// Addresses include those from both the machine and devices.
			// Sorted by scope - public first, then cloud local.
			// Then sorted lexically within the same scope.
			{Addresses: []string{
				"1.1.1.1",
				"9.9.9.9",
				"0.1.2.3",
				"2.2.2.2",
			}},
			{Addresses: []string{
				"10.10.10.10",
				"3.3.3.3",
				"0.3.2.1",
				"4.4.4.4",
			}},
		},
	})
	s.backend.stub.CheckCalls(c, []jujutesting.StubCall{
		{"GetMachineForEntity", []interface{}{s.uOther}},
		{"GetMachineForEntity", []interface{}{s.m0}},
		{"GetMachineForEntity", []interface{}{s.uFoo}},
	})
}

func (s *facadeSuite) TestPublicKeys(c *gc.C) {
	args := params.Entities{
		Entities: []params.Entity{{s.m0}, {s.uOther}, {s.uFoo}},
	}
	results, err := s.facade.PublicKeys(args)

	c.Assert(err, jc.ErrorIsNil)
	c.Check(results, gc.DeepEquals, params.SSHPublicKeysResults{
		Results: []params.SSHPublicKeysResult{
			{PublicKeys: []string{"rsa0", "dsa0"}},
			{Error: apiservertesting.NotFoundError("entity")},
			{PublicKeys: []string{"rsa1", "dsa1"}},
		},
	})
	s.backend.stub.CheckCalls(c, []jujutesting.StubCall{
		{"GetMachineForEntity", []interface{}{s.m0}},
		{"GetSSHHostKeys", []interface{}{names.NewMachineTag("0")}},
		{"GetMachineForEntity", []interface{}{s.uOther}},
		{"GetMachineForEntity", []interface{}{s.uFoo}},
		{"GetSSHHostKeys", []interface{}{names.NewMachineTag("1")}},
	})
}

func (s *facadeSuite) TestProxyTrue(c *gc.C) {
	s.backend.proxySSH = true
	result, err := s.facade.Proxy()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(result.UseProxy, jc.IsTrue)
	s.backend.stub.CheckCalls(c, []jujutesting.StubCall{
		{"ModelConfig", []interface{}{}},
	})
}

func (s *facadeSuite) TestProxyFalse(c *gc.C) {
	s.backend.proxySSH = false
	result, err := s.facade.Proxy()
	c.Assert(err, jc.ErrorIsNil)
	c.Check(result.UseProxy, jc.IsFalse)
	s.backend.stub.CheckCalls(c, []jujutesting.StubCall{
		{"ModelConfig", []interface{}{}},
	})
}

func (s *facadeSuite) TestLeader(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()
	mockReader := NewMockReader(ctrl)
	mockReader.EXPECT().Leaders().Return(map[string]string{"testme": "ubuntu/4", "ubuntu": "ubuntu/5"}, nil)
	facade, err := sshclient.InternalFacade(s.backend, mockReader, s.authorizer, s.callContext)
	c.Assert(err, jc.ErrorIsNil)
	result, err := facade.Leader(params.Entity{Tag: names.NewApplicationTag("ubuntu").String()})
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(result.Error, gc.IsNil)
	c.Assert(result.Result, gc.Equals, "ubuntu/5")
}

type mockBackend struct {
	stub     jujutesting.Stub
	proxySSH bool
}

func (backend *mockBackend) ModelTag() names.ModelTag {
	return names.NewModelTag("deadbeef-2f18-4fd2-967d-db9663db7bea")
}

func (backend *mockBackend) ModelConfig() (*config.Config, error) {
	backend.stub.AddCall("ModelConfig")
	attrs := testing.FakeConfig()
	attrs["proxy-ssh"] = backend.proxySSH
	conf, err := config.New(config.NoDefaults, attrs)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return conf, nil
}

func (backend *mockBackend) GetMachineForEntity(tagString string) (sshclient.SSHMachine, error) {
	backend.stub.AddCall("GetMachineForEntity", tagString)
	switch tagString {
	case names.NewMachineTag("0").String():
		return &mockMachine{
			tag:            names.NewMachineTag("0"),
			publicAddress:  "1.1.1.1",
			privateAddress: "2.2.2.2",
			addresses: network.SpaceAddresses{
				network.NewSpaceAddress("9.9.9.9", network.WithScope(network.ScopePublic)),
			},
			allNetworkAddresses: network.SpaceAddresses{
				network.NewSpaceAddress("0.1.2.3", network.WithScope(network.ScopeCloudLocal)),
				network.NewSpaceAddress("1.1.1.1", network.WithScope(network.ScopePublic)),
				network.NewSpaceAddress("2.2.2.2", network.WithScope(network.ScopeCloudLocal)),
			},
		}, nil
	case names.NewUnitTag("foo/0").String():
		return &mockMachine{
			tag:            names.NewMachineTag("1"),
			publicAddress:  "3.3.3.3",
			privateAddress: "4.4.4.4",
			addresses: network.SpaceAddresses{
				network.NewSpaceAddress("10.10.10.10", network.WithScope(network.ScopePublic)),
			},
			allNetworkAddresses: network.SpaceAddresses{
				network.NewSpaceAddress("0.3.2.1", network.WithScope(network.ScopeCloudLocal)),
				network.NewSpaceAddress("3.3.3.3", network.WithScope(network.ScopePublic)),
				network.NewSpaceAddress("4.4.4.4", network.WithScope(network.ScopeCloudLocal)),
			},
		}, nil
	}
	return nil, errors.NotFoundf("entity")
}

func (backend *mockBackend) GetSSHHostKeys(tag names.MachineTag) (state.SSHHostKeys, error) {
	backend.stub.AddCall("GetSSHHostKeys", tag)
	switch tag {
	case names.NewMachineTag("0"):
		return state.SSHHostKeys{"rsa0", "dsa0"}, nil
	case names.NewMachineTag("1"):
		return state.SSHHostKeys{"rsa1", "dsa1"}, nil
	}
	return nil, errors.New("machine not found")
}

type mockMachine struct {
	tag            names.MachineTag
	publicAddress  string
	privateAddress string

	addresses           network.SpaceAddresses
	allNetworkAddresses network.SpaceAddresses
}

func (m *mockMachine) MachineTag() names.MachineTag {
	return m.tag
}

func (m *mockMachine) PublicAddress() (network.SpaceAddress, error) {
	return network.NewSpaceAddress(m.publicAddress, network.WithScope(network.ScopePublic)), nil
}

func (m *mockMachine) PrivateAddress() (network.SpaceAddress, error) {
	return network.NewSpaceAddress(m.privateAddress, network.WithScope(network.ScopeCloudLocal)), nil
}

func (m *mockMachine) AllDeviceSpaceAddresses() (network.SpaceAddresses, error) {
	return m.allNetworkAddresses, nil
}

func (m *mockMachine) Addresses() network.SpaceAddresses {
	return m.addresses
}
