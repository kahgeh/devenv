/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

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
package cmd

import (
	"runtime/debug"

	"github.com/kahgeh/devenv/logger"
	"github.com/spf13/cobra"
)

// deInitCmd represents the reset command
var teardownCmd = &cobra.Command{
	Use:   "teardown",
	Short: "Tear down the environment",
	Long:  `Tear down the environment.`,
	Run:   tearDown,
}

func tearDown(_ *cobra.Command, _ []string) {
	startLog()
	log := logger.New()
	defer func() {
		if err := recover(); err != nil {
			log.Debug(err)
			log.Debugf("stacktrace : \n %v" + string(debug.Stack()))
		}
	}()
	createSession().Delete()
}

func init() {
	rootCmd.AddCommand(teardownCmd)
}
