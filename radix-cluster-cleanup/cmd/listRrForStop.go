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
	"fmt"
	"github.com/equinor/radix-cluster-cleanup/pkg/settings"
	"time"

	"github.com/spf13/cobra"
)

var listRrsForStopContinuouslyCommand = &cobra.Command{
	Use:   "list-rrs-for-stop-continuously",
	Short: "Continuously list RadixRegistrations which qualify for stop",
	Long:  "Continuously list RadixRegistrations which qualify for stop",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runFunctionPeriodically(listRrsForStop)
	},
}

var listRrsForStopCommand = &cobra.Command{
	Use:   "list-rrs-for-stop",
	Short: "Lists RadixRegistrations which qualify for stop",
	Long:  "Lists RadixRegistrations which qualify for stop.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listRrsForStop()
	},
}

func listRrsForStop() error {
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
	tooInactiveRrs, err := getTooInactiveRrs(kubeClient, inactivityBeforeStop, action)
	if err != nil {
		return err
	}
	for _, rr := range tooInactiveRrs {
		fmt.Printf("%s\n", rr.Name)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(listRrsForStopCommand)
	rootCmd.AddCommand(listRrsForStopContinuouslyCommand)
}
