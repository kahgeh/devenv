/*
Copyright © 2020 kahgeh

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
	"bufio"
	"fmt"
	"github.com/kahgeh/devenv/cmd/types"
	"github.com/kahgeh/devenv/utils"
	"io"
	"net/http"
	"os"

	"github.com/kahgeh/devenv/fixed"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/provider"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "devenv",
	Short: "Tool to deploy apps to personal dev environments",
	Long: `
		- init creates 
			- vpc 
			- spotfleet
			- ecs cluster
		- start
			- spotfleet target instance count to 1
			- create public ip and attach to ecs instance
			- schedule shutdown, spotfleet target instance count to 0
			- renews certificate
			- front proxy + ecs discovery			
		- stop 
			- spotfleet set to 0, remove public ip, 
		- deploy 
			- httpapi (will start if not started and init if not available`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var loglevelMap = map[string]logger.LogLevel{
	"info":  logger.DetailedLogLevel,
	"debug": logger.DebugLogLevel,
}

func startLog() {
	loglevelChoice := viper.GetString("loglevel")
	var loglevel logger.LogLevel = logger.NormalLogLevel
	if len(loglevelChoice) > 0 {
		loglevel = loglevelMap[loglevelChoice]
	}
	logger.CreateLogger(loglevel)
}

func download(baseUrl string, fileName string, outputPath string) bool {
	log := logger.New()
	defer log.LogDone()
	sourceUrl := fmt.Sprintf("%s/aws/%s", baseUrl, fileName)
	log.Debugf("url=%s", sourceUrl)
	response, err := http.Get(sourceUrl)
	if err != nil {
		log.Failf("an error occured attempting to download %s", sourceUrl)
		return false
	}
	defer utils.CloseReadCloser(response.Body, func(s string) { log.Debug(s) })
	configPath := fixed.GetConfigFolderPath()
	filepath := fmt.Sprintf("%v/%v", configPath, outputPath)
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		log.Failf("an error occured attempting to save %v \n %v", outputPath, err)
		return false
	}
	defer utils.CloseReadCloser(out, func(s string) { log.Debug(s) })

	// Write the body to file
	_, err = io.Copy(out, response.Body)
	log.Infof("downloaded %v", fileName)
	return true
}

func createSession() provider.Session {
	log := logger.New()
	defer log.LogDone()
	providerName := provider.Aws
	session, err := provider.NewSession(providerName)
	if err != nil {
		log.Failf("Cannot start provider(%v) session ", providerName)
		return nil
	}
	return session
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cfgFilePath := cfgFile
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		cfgFilePath = fmt.Sprintf("%s/config.yaml", fixed.GetConfigFolderPath())
		viper.SetConfigFile(cfgFilePath)
	}

	viper.AutomaticEnv()

	if _, err := os.Stat(cfgFilePath); os.IsNotExist(err) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Provide the following details:")
		fmt.Print("HostedZoneName: ")
		hostedZoneName, consoleReadErr := reader.ReadString('\n')
		if consoleReadErr != nil {
			fmt.Println("failed to read user setting entries")
			os.Exit(logger.ExitFailureStatus)
		}
		viper.Set("hosted-zone-name", hostedZoneName)

		fmt.Print("DomainName: ")
		domainName, consoleReadErr := reader.ReadString('\n')
		if consoleReadErr != nil {
			fmt.Println("failed to read user setting entries")
			os.Exit(logger.ExitFailureStatus)
		}
		viper.Set("domain-name", domainName)

		fmt.Print("DomainEmail: ")
		domainEmail, consoleReadErr := reader.ReadString('\n')
		if consoleReadErr != nil {
			fmt.Println("failed to read user setting entries")
			os.Exit(logger.ExitFailureStatus)
		}
		viper.Set("domain-email", domainEmail)

		fmt.Print("Environment name: ")
		envName, consoleReadErr := reader.ReadString('\n')
		if consoleReadErr != nil {
			fmt.Println("failed to read user setting entries")
			os.Exit(logger.ExitFailureStatus)
		}
		viper.Set("env-name", envName)

		err := viper.WriteConfigAs(cfgFilePath)
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		fmt.Print(err.Error())
	}
}

func init() {

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/config.yaml)")
	rootCmd.PersistentFlags().String("loglevel", "", "info or debug")
	rootCmd.PersistentFlags().String(string(types.ArgDomainName), "", "--domain-name <app.xyz.com>")
	rootCmd.PersistentFlags().String(string(types.ArgDomainEmail), "", "--domain-email <ibu@xyz.com>")
	rootCmd.PersistentFlags().String(string(types.ArgEnvName), "DevTest", "--env-name DevTest")

	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		fmt.Printf("fail to bind arguments to command \n %s", err.Error())
		os.Exit(logger.ExitFailureStatus)
	}

}
