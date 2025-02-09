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
	"time"

	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"github.com/equinor/radix-operator/pkg/apis/kube"
	v1 "github.com/equinor/radix-operator/pkg/apis/radix/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var deleteRrsContinuouslyCommand = &cobra.Command{
	Use:   "delete-inactive-rrs-continuously",
	Short: "Continuously delete inactive RadixRegistrations",
	Long:  "Continuously delete inactive RadixRegistrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFunctionPeriodically(cmd.Context(), deleteRrs)
	},
}

var deleteRrsCommand = &cobra.Command{
	Use:   "delete-inactive-rrs",
	Short: "Delete inactive RadixRegistrations",
	Long:  "Delete inactive RadixRegistrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		return deleteRrs(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(deleteRrsCommand)
	rootCmd.AddCommand(deleteRrsContinuouslyCommand)
}

func deleteRrs(ctx context.Context) error {
	kubeClient, err := getKubeUtil()
	if err != nil {
		return err
	}
	action := "deletion"
	inactiveDaysBeforeDeletion, err := rootCmd.Flags().GetInt64(settings.InactiveDaysBeforeDeletionOption)
	if err != nil {
		return err
	}
	inactivityBeforeDeletion := time.Hour * 24 * time.Duration(inactiveDaysBeforeDeletion)
	tooInactiveRrs, err := getTooInactiveRrs(ctx, kubeClient, inactivityBeforeDeletion, action)
	if err != nil {
		return err
	}
	for _, rr := range tooInactiveRrs {
		err := deleteRr(ctx, kubeClient, rr)
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteRr(ctx context.Context, client *kube.Kube, rr v1.RadixRegistration) error {
	err := client.RadixClient().RadixV1().RadixRegistrations().Delete(ctx, rr.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Info().Str("appName", rr.Name).Msg("Deleted RadixRegistration")
	return nil
}
