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
	"github.com/kahgeh/devenv/fixed"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/provider/types"
	"github.com/kahgeh/devenv/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"runtime/debug"
)

const (
	baseUrl string = "https://raw.githubusercontent.com/kahgeh/devenv/master"
)

var cloudProviderFiles = map[string][]string{"aws": {"vpc.yml", "ecsCluster.yml",
	"spotFleet.yml", "publicIp.yml",
	"ecr.yml", "app.yml", "front-proxy/app.yml", "front-proxy/Dockerfile", "front-proxy/init-and-run.sh", "front-proxy/envoy-template.py"}}

func initialiseConfig() {
	log := logger.NewTaskLogger()
	defer log.LogDone()

	configFolderPath := fixed.GetConfigFolderPath()
	log.Infof("ensuring config folder '%s' exists...", configFolderPath)
	err := utils.CreateFolderIfNotExist(configFolderPath)
	if err != nil {
		log.Debug(err.Error())
		log.Failf("fail to ensure %q exist", configFolderPath)
	}
	cloudProvider := "aws"
	cloudProviderFolderPath := fmt.Sprintf("%v/%v", configFolderPath, cloudProvider)
	log.Infof("ensuring provider config folder '%s' exists...", cloudProviderFolderPath)
	err = utils.CreateFolderIfNotExist(cloudProviderFolderPath)
	if err != nil {
		log.Debug(err.Error())
		log.Failf("fail to ensure %q exist", cloudProviderFolderPath)
	}
	frontProxyFolderPath :=fmt.Sprintf("%v/front-proxy", cloudProviderFolderPath)
	log.Infof("ensuring front proxy folder '%s' exists...", frontProxyFolderPath)
	err = utils.CreateFolderIfNotExist(frontProxyFolderPath)
	if err != nil {
		log.Debug(err.Error())
		log.Failf("fail to ensure %q exist", frontProxyFolderPath)
	}

	log.Info("downloading cloud provider files...")
	for _, fileName := range cloudProviderFiles[cloudProvider] {
		cloudProviderFilePath := fmt.Sprintf("%v/%v", cloudProvider, fileName)
		download(baseUrl, fileName, cloudProviderFilePath)
	}
	log.Succeed()
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [config]",
	Short: "initialises a vpc, spotfleet and an ecs cluster",
	Long:  `initialises infrastructure, does not cost to keep it`,
	Run:   initialise,
}

func initialise(_ *cobra.Command, args []string) {
	startLog()
	log := logger.New()
	defer log.LogDone()
	defer func() {
		if err := recover(); err != nil {
			log.Failf("%v", err)
			log.Debugf("stacktrace : \n %v" + string(debug.Stack()))
		}
	}()

	if len(args) == 1 && args[0] == "config" {
		initialiseConfig()
		return
	}
	domainName := viper.GetString("domain-name")
	domainEmail := viper.GetString("domain-email")
	envName := viper.GetString("env-name")
	createSession().Initialise(&types.InitialisationParameters{
		DomainName:  domainName,
		DomainEmail: domainEmail,
		EnvironmentName: envName,
	})
}

func init() {
	rootCmd.AddCommand(initCmd)
}
