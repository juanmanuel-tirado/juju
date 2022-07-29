// Copyright 2022 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package modelupgrader

import (
	"fmt"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/names/v4"
	"github.com/juju/version/v2"

	"github.com/juju/juju/apiserver/common"
	apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/apiserver/facade"
	"github.com/juju/juju/core/permission"
	"github.com/juju/juju/docker"
	"github.com/juju/juju/docker/registry"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/bootstrap"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/context"
	"github.com/juju/juju/rpc/params"
	"github.com/juju/juju/upgrades/upgradevalidation"
)

var logger = loggo.GetLogger("juju.apiserver.modelupgrader")

// ModelUpgraderAPI implements the model upgrader interface and is
// the concrete implementation of the api end point.
type ModelUpgraderAPI struct {
	statePool   StatePool
	check       common.BlockCheckerInterface
	authorizer  facade.Authorizer
	toolsFinder common.ToolsFinder
	apiUser     names.UserTag
	isAdmin     bool
	callContext context.ProviderCallContext
	newEnviron  common.NewEnvironFunc

	registryAPIFunc func(repoDetails docker.ImageRepoDetails) (registry.Registry, error)
}

// NewModelUpgraderAPI creates a new api server endpoint for managing
// models.
func NewModelUpgraderAPI(
	controllerTag names.ControllerTag,
	stPool StatePool,
	toolsFinder common.ToolsFinder,
	newEnviron common.NewEnvironFunc,
	blockChecker common.BlockCheckerInterface,
	authorizer facade.Authorizer,
	callCtx context.ProviderCallContext,
	registryAPIFunc func(docker.ImageRepoDetails) (registry.Registry, error),
) (*ModelUpgraderAPI, error) {
	if !authorizer.AuthClient() {
		return nil, apiservererrors.ErrPerm
	}
	// Since we know this is a user tag (because AuthClient is true),
	// we just do the type assertion to the UserTag.
	apiUser, _ := authorizer.GetAuthTag().(names.UserTag)

	isAdmin, err := authorizer.HasPermission(permission.SuperuserAccess, controllerTag)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &ModelUpgraderAPI{
		statePool:       stPool,
		check:           blockChecker,
		authorizer:      authorizer,
		toolsFinder:     toolsFinder,
		apiUser:         apiUser,
		isAdmin:         isAdmin,
		callContext:     callCtx,
		newEnviron:      newEnviron,
		registryAPIFunc: registryAPIFunc,
	}, nil
}

func (m *ModelUpgraderAPI) hasWriteAccess(modelTag names.ModelTag) (bool, error) {
	canWrite, err := m.authorizer.HasPermission(permission.WriteAccess, modelTag)
	if errors.Is(err, errors.NotFound) {
		return false, nil
	}
	return canWrite, err
}

// ConfigSource describes a type that is able to provide config.
// Abstracted primarily for testing.
type ConfigSource interface {
	Config() (*config.Config, error)
}

// AbortModelUpgrade aborts and archives the model upgrade
// synchronisation record, if any.
func (m *ModelUpgraderAPI) AbortModelUpgrade(arg params.ModelParam) error {
	modelTag, err := names.ParseModelTag(arg.ModelTag)
	if err != nil {
		return errors.Trace(err)
	}
	if canWrite, err := m.hasWriteAccess(modelTag); err != nil {
		return errors.Trace(err)
	} else if !canWrite && !m.isAdmin {
		return apiservererrors.ErrPerm
	}

	if err := m.check.ChangeAllowed(); err != nil {
		return errors.Trace(err)
	}
	st, err := m.statePool.Get(modelTag.Id())
	if err != nil {
		return errors.Trace(err)
	}
	defer st.Release()
	return st.AbortCurrentUpgrade()
}

// UpgradeModel upgrades a model.
func (m *ModelUpgraderAPI) UpgradeModel(arg params.UpgradeModelParams) (result params.UpgradeModelResult, err error) {
	logger.Tracef("UpgradeModel arg %#v", arg)
	targetVersion := arg.TargetVersion
	defer func() {
		if err == nil {
			result.ChosenVersion = targetVersion
		}
	}()

	modelTag, err := names.ParseModelTag(arg.ModelTag)
	if err != nil {
		return result, errors.Trace(err)
	}
	if canWrite, err := m.hasWriteAccess(modelTag); err != nil {
		return result, errors.Trace(err)
	} else if !canWrite && !m.isAdmin {
		return result, apiservererrors.ErrPerm
	}

	if err := m.check.ChangeAllowed(); err != nil {
		return result, errors.Trace(err)
	}

	// We now need to access the state pool for that given model.
	st, err := m.statePool.Get(modelTag.Id())
	if err != nil {
		return result, errors.Trace(err)
	}
	defer st.Release()

	model, err := st.Model()
	if err != nil {
		return result, errors.Trace(err)
	}

	agentVersion, err := model.AgentVersion()
	if err != nil {
		return result, errors.Trace(err)
	}
	targetVersion, err = m.decideVersion(
		targetVersion, agentVersion, arg.AgentStream, st, model,
	)
	if errors.Is(errors.Cause(err), errors.NotFound) || errors.Is(errors.Cause(err), errors.AlreadyExists) {
		result.Error = apiservererrors.ServerError(err)
		return result, nil
	}

	if err != nil {
		return result, errors.Trace(err)
	}

	// Before changing the agent version to trigger an upgrade or downgrade,
	// we'll do a very basic check to ensure the environment is accessible.
	envOrBroker, err := m.newEnviron()
	if err != nil {
		return result, errors.Trace(err)
	}
	if err := preCheckEnvironForUpgradeModel(
		m.callContext, envOrBroker, model, agentVersion, targetVersion,
	); err != nil {
		return result, errors.Trace(err)
	}

	if err := m.validateModelUpgrade(false, modelTag, targetVersion, st, model); err != nil {
		return result, errors.Trace(err)
	}
	if arg.DryRun {
		return result, nil
	}

	var agentStream *string
	if arg.AgentStream != "" {
		agentStream = &arg.AgentStream
	}
	if err := st.SetModelAgentVersion(targetVersion, agentStream, arg.IgnoreAgentVersions); err != nil {
		return result, errors.Trace(err)
	}
	return result, nil
}

func preCheckEnvironForUpgradeModel(
	ctx context.ProviderCallContext, env environs.BootstrapEnviron,
	model Model, agentVersion, targetVersion version.Number,
) error {
	if err := environs.CheckProviderAPI(env, ctx); err != nil {
		return errors.Trace(err)
	}

	if model.Name() != bootstrap.ControllerModelName {
		return nil
	}

	precheckEnv, ok := env.(environs.JujuUpgradePrechecker)
	if !ok {
		return nil
	}

	// skipTarget returns true if the from version is less than the target version
	// AND the target version is greater than the to version.
	// Borrowed from upgrades.opsIterator.
	skipTarget := func(from, target, to version.Number) bool {
		// Clear the version tag of the to release to ensure that all
		// upgrade steps for the release are run for alpha and beta
		// releases.
		// ...but only do this if the from version has actually changed,
		// lest we trigger upgrade mode unnecessarily for non-final
		// versions.
		if from.Compare(to) != 0 {
			to.Tag = ""
		}
		// Do not run steps for versions of Juju earlier or same as we are upgrading from.
		if target.Compare(from) <= 0 {
			return true
		}
		// Do not run steps for versions of Juju later than we are upgrading to.
		if target.Compare(to) > 0 {
			return true
		}
		return false
	}

	if err := precheckEnv.PreparePrechecker(); err != nil {
		return err
	}

	for _, op := range precheckEnv.PrecheckUpgradeOperations() {
		if skipTarget(agentVersion, op.TargetVersion, targetVersion) {
			logger.Debugf("ignoring precheck upgrade operation for version %s", op.TargetVersion)
			continue
		}
		logger.Debugf("running precheck upgrade operation for version %s", op.TargetVersion)
		for _, step := range op.Steps {
			logger.Debugf("running precheck step %q", step.Description())
			if err := step.Run(); err != nil {
				return errors.Annotatef(err, "Unable to upgrade to %s:", targetVersion)
			}
		}
	}
	return nil
}

func (m *ModelUpgraderAPI) validateModelUpgrade(
	force bool, modelTag names.ModelTag, targetVersion version.Number,
	st State, model Model,
) (err error) {
	var blockers *upgradevalidation.ModelUpgradeBlockers
	defer func() {
		if err == nil && blockers != nil {
			err = apiservererrors.ServerError(
				errors.NewNotSupported(nil,
					fmt.Sprintf(
						"cannot upgrade to %q due to issues with these models:\n%s",
						targetVersion, blockers,
					),
				),
			)
		}
	}()

	isControllerModel := model.IsControllerModel()
	if !isControllerModel {
		validators := upgradevalidation.ValidatorsForModelUpgrade(force, targetVersion)
		checker := upgradevalidation.NewModelUpgradeCheck(modelTag.Id(), m.statePool, st, model, validators...)
		blockers, err = checker.Validate()
		if err != nil {
			return errors.Trace(err)
		}
		return
	}

	checker := upgradevalidation.NewModelUpgradeCheck(
		modelTag.Id(), m.statePool, st, model,
		upgradevalidation.ValidatorsForControllerUpgrade(true, targetVersion)...,
	)
	blockers, err = checker.Validate()
	if err != nil {
		return errors.Trace(err)
	}

	modelUUIDs, err := st.AllModelUUIDs()
	if err != nil {
		return errors.Trace(err)
	}
	validators := upgradevalidation.ValidatorsForControllerUpgrade(false, targetVersion)
	for _, modelUUID := range modelUUIDs {
		if modelUUID == modelTag.Id() {
			// We have done checks for controller model above already.
			continue
		}
		st, err := m.statePool.Get(modelUUID)
		if err != nil {
			return errors.Trace(err)
		}
		defer st.Release()
		model, err := st.Model()
		if err != nil {
			return errors.Trace(err)
		}
		checker := upgradevalidation.NewModelUpgradeCheck(modelUUID, m.statePool, st, model, validators...)
		blockersForModel, err := checker.Validate()
		if err != nil {
			return errors.Trace(err)
		}
		if blockersForModel == nil {
			// all good.
			continue
		}
		if blockers == nil {
			blockers = blockersForModel
			continue
		}
		blockers.Join(blockersForModel)
	}
	return
}
