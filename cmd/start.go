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
	"fmt"
	"github.com/kahgeh/devenv/provider/types"
	"os"
	"runtime/debug"

	"github.com/kahgeh/devenv/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: start,
}

func mapParameters(_ []string) *types.StartParameters {
	return &types.StartParameters{
		HostedZoneName: viper.GetString("hosted-zone-name"),
		DomainName:  viper.GetString("domain-name"),
		EnvironmentName: viper.GetString("env-name"),
	}
}

func start(_ *cobra.Command, args []string) {
	startLog()
	log := logger.New()
	defer log.LogDone()
	defer func() {
		if err := recover(); err != nil {
			log.Failf("%v", err)
			log.Debugf("stacktrace : \n %v" + string(debug.Stack()))
		}
	}()
	parameters := mapParameters(args)
	createSession().Start(parameters)
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.PersistentFlags().String("hosted-zone-name", "", "--hosted-zone-name xyz.com. , remember to include the period at the end")
	err := viper.BindPFlags(deployCmd.PersistentFlags())
	if err != nil {
		fmt.Printf("fail to bind command arguments\n %s", err.Error())
		os.Exit(logger.ExitFailureStatus)
	}
}
