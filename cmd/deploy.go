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
	cmdTypes "github.com/kahgeh/devenv/cmd/types"
	"github.com/kahgeh/devenv/provider/types"
	"os"
	"runtime/debug"

	"github.com/kahgeh/devenv/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	argPath    cmdTypes.ArgName = "path"
	argAppType cmdTypes.ArgName = "type"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy <name>",
	Short: "deploy service",
	Long: `deploy service
when
	type is front-proxy, deploy will recreate the stack, also it will use http ports 80, 443`,
	Run: deploy,
}

func extractParameters(args []string) (appName string, appType types.AppType, path string, envName string, domainName string, domainEmail string, err error) {
	appType = types.AppType(viper.GetString(string(argAppType)))

	if appType == types.FrontProxy {
		appName = string(cmdTypes.KnownAppFrontProxy)
	} else {

		if len(args) < 1 {
			err = &cmdTypes.MissingArgument{
				ParameterName: "appName",
			}
			return
		}
		appName = args[0]
	}
	path = viper.GetString(string(argPath))
	envName = viper.GetString(string(cmdTypes.ArgEnvName))
	domainName = viper.GetString(string(cmdTypes.ArgDomainName))
	domainEmail = viper.GetString(string(cmdTypes.ArgDomainEmail))

	return
}

func deploy(_ *cobra.Command, args []string) {
	startLog()
	log := logger.New()
	defer log.LogDone()
	defer func() {
		if err := recover(); err != nil {
			log.Failf("%v", err)
			log.Debugf("stacktrace : \n %v" + string(debug.Stack()))
		}
	}()

	appName, appType, path, envName, domainName, domainEmail, err := extractParameters(args)
	if err != nil {
		log.Fail(err.Error())
	}

	createSession().Deploy(&types.DeployParameters{
		AppType:         appType,
		AppName:         appName,
		Path:            path,
		DomainName:      domainName,
		DomainEmail:     domainEmail,
		EnvironmentName: envName,
	})
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.PersistentFlags().String(string(argPath), ".", "--path <relative or absolute path>")
	deployCmd.PersistentFlags().String(string(argAppType), "api", "--type [default is api - other options include front-proxy]")
	err := viper.BindPFlags(deployCmd.PersistentFlags())
	if err != nil {
		fmt.Printf("fail to bind command arguments\n %s", err.Error())
		os.Exit(logger.ExitFailureStatus)
	}
}
