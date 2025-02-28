/*
Copyright 2016 The Rook Authors. All rights reserved.

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

package ceph

import (
	"github.com/pkg/errors"
	"github.com/rook/rook/cmd/rook/rook"
	"github.com/rook/rook/pkg/daemon/ceph/agent/flexvolume/attachment"
	operator "github.com/rook/rook/pkg/operator/ceph"
	cluster "github.com/rook/rook/pkg/operator/ceph/cluster"
	"github.com/rook/rook/pkg/operator/ceph/cluster/mon"
	"github.com/rook/rook/pkg/operator/ceph/csi"

	opcontroller "github.com/rook/rook/pkg/operator/ceph/controller"
	"github.com/rook/rook/pkg/operator/k8sutil"
	"github.com/rook/rook/pkg/util/flags"
	"github.com/spf13/cobra"
)

const (
	containerName = "rook-ceph-operator"
)

var operatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Runs the Ceph operator for orchestrating and managing Ceph storage in a Kubernetes cluster",
	Long: `Runs the Ceph operator for orchestrating and managing Ceph storage in a Kubernetes cluster
https://github.com/rook/rook`,
}

func init() {
	operatorCmd.Flags().DurationVar(&mon.HealthCheckInterval, "mon-healthcheck-interval", mon.HealthCheckInterval, "mon health check interval (duration)")
	operatorCmd.Flags().DurationVar(&mon.MonOutTimeout, "mon-out-timeout", mon.MonOutTimeout, "mon out timeout (duration)")

	// csi deployment templates
	operatorCmd.Flags().StringVar(&csi.RBDPluginTemplatePath, "csi-rbd-plugin-template-path", csi.DefaultRBDPluginTemplatePath, "path to ceph-csi rbd plugin template")

	operatorCmd.Flags().StringVar(&csi.RBDProvisionerDepTemplatePath, "csi-rbd-provisioner-dep-template-path", csi.DefaultRBDProvisionerDepTemplatePath, "path to ceph-csi rbd provisioner deployment template")

	operatorCmd.Flags().StringVar(&csi.CephFSPluginTemplatePath, "csi-cephfs-plugin-template-path", csi.DefaultCephFSPluginTemplatePath, "path to ceph-csi cephfs plugin template")
	operatorCmd.Flags().StringVar(&csi.CephFSProvisionerDepTemplatePath, "csi-cephfs-provisioner-dep-template-path", csi.DefaultCephFSProvisionerDepTemplatePath, "path to ceph-csi cephfs provisioner deployment template")

	operatorCmd.Flags().BoolVar(&cluster.EnableMachineDisruptionBudget, "enable-machine-disruption-budget", false, "enable fencing controllers")

	flags.SetFlagsFromEnv(operatorCmd.Flags(), rook.RookEnvVarPrefix)
	flags.SetLoggingFlags(operatorCmd.Flags())
	operatorCmd.RunE = startOperator
}

func startOperator(cmd *cobra.Command, args []string) error {

	rook.SetLogLevel()

	rook.LogStartupInfo(operatorCmd.Flags())

	logger.Info("starting Rook-Ceph operator")
	context := createContext()
	context.ConfigDir = k8sutil.DataDir
	volumeAttachment, err := attachment.New(context)
	if err != nil {
		rook.TerminateFatal(err)
	}

	rook.CheckOperatorResources(context.Clientset)
	rookImage := rook.GetOperatorImage(context.Clientset, containerName)
	rookBaseImageCephVersion, err := rook.GetOperatorBaseImageCephVersion(context)
	if err != nil {
		logger.Errorf("failed to get operator base image ceph version. %v", err)
	}
	opcontroller.OperatorCephBaseImageVersion = rookBaseImageCephVersion
	logger.Infof("base ceph version inside the rook operator image is %q", opcontroller.OperatorCephBaseImageVersion)

	serviceAccountName := rook.GetOperatorServiceAccount(context.Clientset)
	op := operator.New(context, volumeAttachment, rookImage, serviceAccountName)
	err = op.Run()
	if err != nil {
		rook.TerminateFatal(errors.Wrap(err, "failed to run operator\n"))
	}

	return nil
}
