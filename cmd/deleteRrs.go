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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var deleteRrs = &cobra.Command{
	Use:   "delete-inactive-rrs",
	Short: "Delete inactive RadixRegistrations",
	Long:  "Delete inactive RadixRegistrations",
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeClient, err := getKubeUtil()
		if err != nil {
			return err
		}
		action := "deletion"
		inactiveDaysBeforeDeletion, err := rootCmd.Flags().GetInt64(settings.InactiveDaysBeforeDeletionOption)
		inactivityBeforeDeletion := time.Hour * 24 * time.Duration(inactiveDaysBeforeDeletion)
		tooInactiveRrs, err := getTooInactiveRrs(kubeClient, inactivityBeforeDeletion, action)
		if err != nil {
			return err
		}
		for _, rr := range tooInactiveRrs {
			err := deleteRr(kubeClient, rr)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteRrs)
}

func deleteRr(client *kube.Kube, rr v1.RadixRegistration) error {
	err := client.RadixClient().RadixV1().RadixRegistrations().Delete(context.TODO(), rr.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	log.Infof("Deleted RadixRegistration %s", rr.Name)
	return nil
}
