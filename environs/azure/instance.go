// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package azure

import (
	"fmt"

	"launchpad.net/gwacl"

	"launchpad.net/juju-core/environs"
	"launchpad.net/juju-core/instance"
)

type azureInstance struct {
	// An instance contains an Azure Service (instance==service).
	gwacl.HostedServiceDescriptor
	environ *azureEnviron
}

// azureInstance implements Instance.
var _ instance.Instance = (*azureInstance)(nil)

// Id is specified in the Instance interface.
func (azInstance *azureInstance) Id() instance.Id {
	return instance.Id(azInstance.ServiceName)
}

var AZURE_DOMAIN_NAME = "cloudapp.net"

// Addresses is specified in the Instance interface.
func (azInstance *azureInstance) Addresses() ([]instance.Address, error) {
	logger.Errorf("azureInstance.Addresses not implemented")
	return nil, nil
}

// DNSName is specified in the Instance interface.
func (azInstance *azureInstance) DNSName() (string, error) {
	// For deployments in the Production slot, the instance's DNS name
	// is its service name, in the cloudapp.net domain.
	// (For Staging deployments it's all much weirder: they get random
	// names assigned, which somehow don't seem to resolve from the
	// outside.)
	name := fmt.Sprintf("%s.%s", azInstance.ServiceName, AZURE_DOMAIN_NAME)
	return name, nil
}

// WaitDNSName is specified in the Instance interface.
func (azInstance *azureInstance) WaitDNSName() (string, error) {
	return environs.WaitDNSName(azInstance)
}

// OpenPorts is specified in the Instance interface.
func (azInstance *azureInstance) OpenPorts(machineId string, ports []instance.Port) error {
	env := azInstance.environ
	context, err := env.getManagementAPI()
	if err != nil {
		return err
	}
	defer env.releaseManagementAPI(context)

	env.Lock()
	defer env.Unlock()

	deployments, err := context.ListAllDeployments(&gwacl.ListAllDeploymentsRequest{
		ServiceName: azInstance.ServiceName,
	})
	if err != nil {
		return err
	}

	for _, deployment := range deployments {
		for _, role := range deployment.RoleList {
			request := &gwacl.AddRoleEndpointsRequest{
				ServiceName:    azInstance.ServiceName,
				DeploymentName: deployment.Name,
				RoleName:       role.RoleName,
			}
			for _, port := range ports {
				request.InputEndpoints = append(
					request.InputEndpoints, gwacl.InputEndpoint{
						LocalPort: port.Number,
						Name:      fmt.Sprintf("%s%d", port.Protocol, port.Number),
						Port:      port.Number,
						Protocol:  port.Protocol,
					})
			}
			err := context.AddRoleEndpoints(request)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ClosePorts is specified in the Instance interface.
func (azInstance *azureInstance) ClosePorts(machineId string, ports []instance.Port) error {
	// TODO: implement this.
	return nil
}

// Ports is specified in the Instance interface.
func (azInstance *azureInstance) Ports(machineId string) ([]instance.Port, error) {
	// TODO: implement this.
	return []instance.Port{}, nil
}
