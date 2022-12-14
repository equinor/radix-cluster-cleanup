// Copyright © 2022
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/equinor/radix-operator/pkg/apis/utils/numbers"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

var stopRrs = &cobra.Command{
	Use:   "stop-inactive-rrs",
	Short: "Stop all components in inactive RadixRegistrations",
	Long:  "Stop all components in inactive RadixRegistrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeClient, err := getKubeUtil()
		if err != nil {
			return err
		}
		action := "stop"
		inactiveDaysBeforeStop, err := rootCmd.Flags().GetInt64(settings.InactiveDaysBeforeStopOption)
		inactivityBeforeStop := time.Hour * 24 * time.Duration(inactiveDaysBeforeStop)
		tooInactiveRrs, err := getTooInactiveRrs(kubeClient, inactivityBeforeStop, action)
		if err != nil {
			return err
		}
		for _, rr := range tooInactiveRrs {
			err := stopRr(kubeClient, rr)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func stopRr(kubeClient *kube.Kube, rr v1.RadixRegistration) error {
	ra, err := getRadixApplication(kubeClient, rr.Name)
	if err != nil {
		return err
	}
	namespaces := getRuntimeNamespaces(ra)
	rdsForRr, err := getRadixDeploymentsInNamespaces(kubeClient, namespaces)
	for _, rd := range rdsForRr {
		isActive := rdIsActive(rd)
		if err != nil {
			return err
		}
		if isActive {
			err := scaleRdComponentsToZeroReplicas(kubeClient, rd)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func scaleRdComponentsToZeroReplicas(kubeClient *kube.Kube, rd v1.RadixDeployment) error {
	componentNames := make([]string, 0)
	for i, _ := range rd.Spec.Components {
		rd.Spec.Components[i].Replicas = numbers.IntPtr(0)
		componentNames = append(componentNames, rd.Spec.Components[i].Name)
	}
	_, err := kubeClient.RadixClient().RadixV1().RadixDeployments(rd.Namespace).Update(context.TODO(), &rd, metav1.UpdateOptions{})
	log.Infof("scaled component %s in rd %s to 0 replicas", strings.Join(componentNames, ", "), rd.Name)
	if err != nil {
		return err
	}
	return nil
}

func rdIsActive(rd v1.RadixDeployment) bool {
	return rd.Status.Condition == v1.DeploymentActive
}

func init() {
	rootCmd.AddCommand(stopRrs)
}
