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
	"github.com/spf13/cobra"
)

var listRrsForStopAndDeletionContinuouslyCommand = &cobra.Command{
	Use:   "list-rrs-for-stop-and-deletion-continuously",
	Short: "Continuously list RadixRegistrations which qualify for stop and deletion",
	Long:  "Continuously list RadixRegistrations which qualify for stop and deletion",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFunctionPeriodically(listRrsForStopAndDeletion)
	},
}

func listRrsForStopAndDeletion() error {
	err := listRrsForStop()
	if err != nil {
		return err
	}
	return listRrsForDeletion()
}

func init() {
	rootCmd.AddCommand(listRrsForStopAndDeletionContinuouslyCommand)
}
