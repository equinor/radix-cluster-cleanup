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
	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"time"

	"github.com/spf13/cobra"
)

var listRrsForDeletion = &cobra.Command{
	Use:   "list-rrs-for-deletion",
	Short: "Lists RadixRegistrations which qualify for deletion",
	Long:  "Lists RadixRegistrations which qualify for deletion.",
	RunE: func(cmd *cobra.Command, args []string) error {
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
		tooInactiveRrs, err := getTooInactiveRrs(kubeClient, inactivityBeforeDeletion, action)
		if err != nil {
			return err
		}
		for _, rr := range tooInactiveRrs {
			println(rr.Name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listRrsForDeletion)
}