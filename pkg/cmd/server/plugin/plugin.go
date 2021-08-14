/*
Copyright 2017, 2019 the Velero contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/vmware-tanzu/velero/pkg/backup"
	"github.com/vmware-tanzu/velero/pkg/client"
	velerodiscovery "github.com/vmware-tanzu/velero/pkg/discovery"
	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
	"github.com/vmware-tanzu/velero/pkg/restore"
)

func NewCommand(f client.Factory) *cobra.Command {
	pluginServer := veleroplugin.NewServer()
	c := &cobra.Command{
		Use:    "run-plugins",
		Hidden: true,
		Short:  "INTERNAL COMMAND ONLY - not intended to be run directly by users",
		Run: func(c *cobra.Command, args []string) {
			pluginServer.
				RegisterBackupItemAction(PVBackupPlugin, newPVBackupItemAction).
				RegisterBackupItemAction(PodBackupPlugin, newPodBackupItemAction).
				RegisterBackupItemAction(ServiceAccountBackupPlugin, newServiceAccountBackupItemAction(f)).
				RegisterBackupItemAction(RemapCRDVersionBackupPlugin, newRemapCRDVersionBackupItemAction(f)).
				RegisterRestoreItemAction(JobRestorePlugin, newJobRestoreItemAction).
				RegisterRestoreItemAction(PodRestorePlugin, newPodRestoreItemAction).
				RegisterRestoreItemAction(ResticRestorePlugin, newResticRestoreItemAction(f)).
				RegisterRestoreItemAction(InitRestoreHookPlugin, newInitRestoreHookPodAction).
				RegisterRestoreItemAction(ServiceRestorePlugin, newServiceRestoreItemAction).
				RegisterRestoreItemAction(ServiceAccountRestorePlugin, newServiceAccountRestoreItemAction).
				RegisterRestoreItemAction(AddPVCFromPodRestorePlugin, newAddPVCFromPodRestoreItemAction).
				RegisterRestoreItemAction(AddPVFromPVCRestorePlugin, newAddPVFromPVCRestoreItemAction).
				RegisterRestoreItemAction(ChangeStorageClassRestorePlugin, newChangeStorageClassRestoreItemAction(f)).
				RegisterRestoreItemAction(RoleBindingItemPlugin, newRoleBindingItemAction).
				RegisterRestoreItemAction(ClusterRoleBindingRestorePlugin, newClusterRoleBindingRestoreItemAction).
				RegisterRestoreItemAction(CRDV1PreserveUnknownFieldsRestorePlugin, newCRDV1PreserveUnknownFieldsRestoreItemAction).
				RegisterRestoreItemAction(ChangePVCNodeSelectorPlugin, newChangePVCNodeSelectorItemAction(f)).
				RegisterRestoreItemAction(APIServiceRestorePlugin, newAPIServiceRestoreItemAction).
				Serve()
		},
	}

	pluginServer.BindFlags(c.Flags())

	return c
}

func newPVBackupItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return backup.NewPVCAction(logger), nil
}

func newPodBackupItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return backup.NewPodAction(logger), nil
}

func newServiceAccountBackupItemAction(f client.Factory) veleroplugin.HandlerInitializer {
	return func(logger logrus.FieldLogger) (interface{}, error) {
		// TODO(ncdc): consider a k8s style WantsKubernetesClientSet initialization approach
		clientset, err := f.KubeClient()
		if err != nil {
			return nil, err
		}

		discoveryHelper, err := velerodiscovery.NewHelper(clientset.Discovery(), logger)
		if err != nil {
			return nil, err
		}

		action, err := backup.NewServiceAccountAction(
			logger,
			backup.NewClusterRoleBindingListerMap(clientset),
			discoveryHelper)
		if err != nil {
			return nil, err
		}

		return action, nil
	}
}

func newRemapCRDVersionBackupItemAction(f client.Factory) veleroplugin.HandlerInitializer {
	return func(logger logrus.FieldLogger) (interface{}, error) {
		config, err := f.ClientConfig()
		if err != nil {
			return nil, err
		}

		client, err := apiextensions.NewForConfig(config)
		if err != nil {
			return nil, err
		}

		return backup.NewRemapCRDVersionAction(logger, client.ApiextensionsV1beta1().CustomResourceDefinitions()), nil
	}
}

func newJobRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewJobAction(logger), nil
}

func newPodRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewPodAction(logger), nil
}

func newInitRestoreHookPodAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewInitRestoreHookPodAction(logger), nil
}

func newResticRestoreItemAction(f client.Factory) veleroplugin.HandlerInitializer {
	return func(logger logrus.FieldLogger) (interface{}, error) {
		client, err := f.KubeClient()
		if err != nil {
			return nil, err
		}

		veleroClient, err := f.Client()
		if err != nil {
			return nil, err
		}

		return restore.NewResticRestoreAction(logger, client.CoreV1().ConfigMaps(f.Namespace()), veleroClient.VeleroV1().PodVolumeBackups(f.Namespace())), nil
	}
}

func newServiceRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewServiceAction(logger), nil
}

func newServiceAccountRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewServiceAccountAction(logger), nil
}

func newAddPVCFromPodRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewAddPVCFromPodAction(logger), nil
}

func newAddPVFromPVCRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewAddPVFromPVCAction(logger), nil
}

func newCRDV1PreserveUnknownFieldsRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewCRDV1PreserveUnknownFieldsAction(logger), nil
}

func newChangeStorageClassRestoreItemAction(f client.Factory) veleroplugin.HandlerInitializer {
	return func(logger logrus.FieldLogger) (interface{}, error) {
		client, err := f.KubeClient()
		if err != nil {
			return nil, err
		}

		return restore.NewChangeStorageClassAction(
			logger,
			client.CoreV1().ConfigMaps(f.Namespace()),
			client.StorageV1().StorageClasses(),
		), nil
	}
}

func newRoleBindingItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewRoleBindingAction(logger), nil
}

func newClusterRoleBindingRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewClusterRoleBindingAction(logger), nil
}

func newChangePVCNodeSelectorItemAction(f client.Factory) veleroplugin.HandlerInitializer {
	return func(logger logrus.FieldLogger) (interface{}, error) {
		client, err := f.KubeClient()
		if err != nil {
			return nil, err
		}

		return restore.NewChangePVCNodeSelectorAction(
			logger,
			client.CoreV1().ConfigMaps(f.Namespace()),
			client.CoreV1().Nodes(),
		), nil
	}
}

func newAPIServiceRestoreItemAction(logger logrus.FieldLogger) (interface{}, error) {
	return restore.NewAPIServiceAction(logger), nil
}
