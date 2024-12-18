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
	"strings"
	"time"

	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"github.com/equinor/radix-common/utils/pointers"
	"github.com/equinor/radix-common/utils/slice"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var stopRrsContinuouslyCommand = &cobra.Command{
	Use:   "stop-inactive-rrs-continuously",
	Short: "Continuously stop all components in inactive RadixRegistrations",
	Long:  "Continuously stop all components in inactive RadixRegistrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFunctionPeriodically(cmd.Context(), stopRrs)
	},
}

var stopRrsCommand = &cobra.Command{
	Use:   "stop-inactive-rrs",
	Short: "Stop all components in inactive RadixRegistrations",
	Long:  "Stop all components in inactive RadixRegistrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return stopRrs(cmd.Context())
	},
}

func stopRrs(ctx context.Context) error {
	kubeClient, err := getKubeUtil()
	if err != nil {
		return err
	}
	action := "stop"
	inactiveDaysBeforeStop, err := rootCmd.Flags().GetInt64(settings.InactiveDaysBeforeStopOption)
	if err != nil {
		return err
	}
	inactivityBeforeStop := time.Hour * 24 * time.Duration(inactiveDaysBeforeStop)
	tooInactiveRrs, err := getTooInactiveRrs(ctx, kubeClient, inactivityBeforeStop, action)
	if err != nil {
		return err
	}

	for _, rr := range tooInactiveRrs {
		ctx := log.Ctx(ctx).With().Str("appName", rr.Name).Logger().WithContext(ctx)
		err := stopRr(ctx, kubeClient, rr)
		if err != nil {
			return err
		}
	}
	return nil
}

func stopRr(ctx context.Context, kubeClient *kube.Kube, rr v1.RadixRegistration) error {
	ra, err := getRadixApplication(ctx, kubeClient, rr.Name)
	if err != nil {
		return err
	}
	namespaces := getRuntimeNamespaces(ra)
	rdsForRr, err := getRadixDeploymentsInNamespaces(ctx, kubeClient, namespaces)
	if err != nil {
		return err
	}

	for _, rd := range slice.FindAll(rdsForRr, rdIsActive) {
		ctx := log.Ctx(ctx).With().Str("deployment", rd.Name).Logger().WithContext(ctx)
		if err := scaleRdComponentsToZeroReplicas(ctx, kubeClient, rd); err != nil {
			return err
		}
	}

	return nil
}

func scaleRdComponentsToZeroReplicas(ctx context.Context, kubeClient *kube.Kube, rd v1.RadixDeployment) error {
	logger := log.Ctx(ctx)
	componentNames := make([]string, 0)
	for i := range rd.Spec.Components {
		rd.Spec.Components[i].ReplicasOverride = pointers.Ptr(0)
		componentNames = append(componentNames, rd.Spec.Components[i].Name)
	}
	_, err := kubeClient.RadixClient().RadixV1().RadixDeployments(rd.Namespace).Update(ctx, &rd, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	logger.Info().Msgf("scaled components %s to 0 replicas", strings.Join(componentNames, ", "))
	return nil
}

func rdIsActive(rd v1.RadixDeployment) bool {
	return rd.Status.Condition == v1.DeploymentActive
}

func init() {
	rootCmd.AddCommand(stopRrsCommand)
	rootCmd.AddCommand(stopRrsContinuouslyCommand)
}
