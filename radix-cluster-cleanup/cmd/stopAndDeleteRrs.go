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

	"github.com/spf13/cobra"
)

var StopAndDeleteRrsContinuouslyCommand = &cobra.Command{
	Use:   "stop-and-delete-rrs-continuously",
	Short: "Continuously stop and delete inactive RRs",
	Long:  "Continuously stop and delete inactive RRs",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFunctionPeriodically(cmd.Context(), stopAndDeleteInactiveRrs)
	},
}

func stopAndDeleteInactiveRrs(ctx context.Context) error {
	err := stopRrs(ctx)
	if err != nil {
		return err
	}
	return deleteRrs(ctx)
}

func init() {
	rootCmd.AddCommand(StopAndDeleteRrsContinuouslyCommand)
}
