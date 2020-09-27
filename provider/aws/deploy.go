package aws

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	ecs2 "github.com/aws/aws-sdk-go-v2/service/ecs"
	cmdTypes "github.com/kahgeh/devenv/cmd/types"
	provideTypes "github.com/kahgeh/devenv/provider/types"
	"github.com/kahgeh/devenv/utils"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/docker/docker/api/types"
	whale "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/kahgeh/devenv/logger"
	"github.com/kahgeh/devenv/utils/ctx"
)

func generateID() string {
	s := fmt.Sprintf("%v", time.Now().UnixNano())
	return s
}

func getAuthToken(authResponse *ecr.GetAuthorizationTokenResponse) string {
	authInfoBytes, _ := base64.StdEncoding.DecodeString(*authResponse.AuthorizationData[0].AuthorizationToken)
	authInfo := strings.Split(string(authInfoBytes), ":")
	auth := struct {
		Username string
		Password string
	}{
		Username: authInfo[0],
		Password: authInfo[1],
	}

	authBytes, _ := json.Marshal(auth)
	return base64.StdEncoding.EncodeToString(authBytes)
}

func buildImage(context string, appName string, id string, buildArgs map[string]*string) *string {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	client, err := whale.NewClientWithOpts()
	if err != nil {
		log.Fail("cannot access docker")
		return nil
	}

	buildCtx, _ := archive.TarWithOptions(context, &archive.TarOptions{})
	tag := fmt.Sprintf("%s:%s", appName, id)
	response, err := client.ImageBuild(ctx.GetContext(),
		buildCtx,
		types.ImageBuildOptions{
			Tags:      []string{tag},
			BuildArgs: buildArgs,
			NoCache:   true,
		})

	if err != nil {
		log.Debug(err.Error())
		log.Fail("build failed")
		return nil
	}

	defer utils.CloseReadCloser(response.Body, func(s string) { log.Debug(s) })
	log.DebugFunc(func() {
		termFd, isTerm := term.GetFdInfo(os.Stderr)
		err := jsonmessage.DisplayJSONMessagesStream(response.Body, os.Stderr, termFd, isTerm, nil)
		if err != nil {
			log.Debug(err.Error())
		}
	}, func() {
		_, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fail(err.Error())
		}
	})
	log.Succeed()
	return &tag
}

func getEcrStackName(appName string) string {
	return fmt.Sprintf("ecr-%s", appName)
}

func (session *Session) createRepository(appName string) *string {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	stackName := getEcrStackName(appName)
	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	stackDescription, err := stack.Describe()
	if stackDescription != nil {
		log.Succeed()
		return stackDescription.Outputs[0].OutputValue
	}

	err = stack.Create("ecr.yml", []cloudformation.Parameter{
		{
			ParameterKey:   aws.String("RepositoryName"),
			ParameterValue: aws.String(appName),
		},
	})
	if err != nil {
		log.Fail("fail to create repository")
	}

	repository := session.getStackOutputValue(stackName, stackName)
	log.Succeed()
	return repository
}

func (session *Session) uploadImage(source *string, repo *string, tag string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	client, err := whale.NewClientWithOpts()
	if err != nil {
		log.Fail("cannot access docker")
		return
	}
	target := fmt.Sprintf("%s:%s", *repo, tag)
	log.Info("tagging image with repo prefix...")
	err = client.ImageTag(ctx.GetContext(), *source, target)
	if err != nil {
		log.Debug(err.Error())
		log.Fail("Fail to tag image")
		return
	}
	err = client.ImageTag(ctx.GetContext(), *source, *repo)
	if err != nil {
		log.Debug(err.Error())
		log.Fail("Fail to tag image with latest")
		return
	}

	log.Info("pushing image to repository...")
	svc := ecr.New(session.config)
	request := svc.GetAuthorizationTokenRequest(&ecr.GetAuthorizationTokenInput{})
	authResponse, err := request.Send(ctx.GetContext())
	authToken := getAuthToken(authResponse)
	response, err := client.ImagePush(
		ctx.GetContext(),
		target,
		types.ImagePushOptions{
			RegistryAuth: authToken,
		})

	response, err = client.ImagePush(
		ctx.GetContext(),
		*repo,
		types.ImagePushOptions{
			RegistryAuth: authToken,
		})

	if err != nil {
		log.Debug(err.Error())
		log.Fail("fail to push to registry")
		return
	}

	if response == nil {
		log.Fail("fail to push to registry, no response")
		return
	}

	defer utils.CloseReadCloser(response, func(s string) { log.Debug(s) })
	log.DebugFunc(func() {
		termFd, isTerm := term.GetFdInfo(os.Stderr)
		err := jsonmessage.DisplayJSONMessagesStream(response, os.Stderr, termFd, isTerm, nil)
		if err != nil {
			log.Debug(err.Error())
			log.Fail("fail to push image to repository")
		}
	}, func() {
		_, err := ioutil.ReadAll(response)
		if err != nil {
			log.Debug(err.Error())
			log.Fail("fail to push image to repository")
		}
	})
	log.Succeed()
}

func (session *Session) deployFrontProxy(image string, envName string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	ecsClusterExportName := config.EcsClusterStackName
	appName := string(cmdTypes.KnownAppFrontProxy)
	paramStoreKeyPath := fmt.Sprintf(string(TemplateParamStoreKeyPath), envName)
	log.Infof("image=%s envName%s", image, envName)

	stackName := fmt.Sprintf("app-%s", appName)
	parameters := []cloudformation.Parameter{
		{
			ParameterKey:   aws.String("AppName"),
			ParameterValue: aws.String(appName),
		},
		{
			ParameterKey:   aws.String("Image"),
			ParameterValue: aws.String(image),
		},
		{
			ParameterKey:   aws.String("EcsClusterExportName"),
			ParameterValue: aws.String(ecsClusterExportName),
		},
		{
			ParameterKey:   aws.String("EnvironmentName"),
			ParameterValue: aws.String(envName),
		},
		{
			ParameterKey:   aws.String("ParamStoreKeyArn"),
			ParameterValue: aws.String(paramStoreKeyPath),
		},
	}
	templateFileName := "front-proxy/app.yml"
	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	stackDescription, err := stack.Describe()
	if stackDescription != nil {
		ssmSession := session.NewSsmSession()
		log.Info("getting app details...")
		cluster, err := ssmSession.GetParamterValue(fmt.Sprintf("/allEnvs/%s/infra/ecs/name", envName))
		if err != nil {
			log.Debug(err.Error())
			log.Fail("fail to get cluster name details")
			return
		}

		serviceArn, err := ssmSession.GetParamterValue(fmt.Sprintf("/allEnvs/%s/apps/%s/serviceArn", envName, appName))
		if err != nil {
			log.Debug(err.Error())
			log.Fail("fail to get service details")
			return
		}

		log.Info("removing app...")
		api := ecs2.New(session.config)
		req := api.UpdateServiceRequest(&ecs2.UpdateServiceInput{
			Cluster:      cluster,
			Service:      serviceArn,
			DesiredCount: aws.Int64(0),
		})
		_, err = req.Send(ctx.GetContext())
		if err != nil {
			log.Debug(err.Error())
			log.Fail("fail to remove app first")
			return
		}
		log.Info("waiting app to be removed...")
		err = api.WaitUntilServicesStable(ctx.GetContext(), &ecs2.DescribeServicesInput{
			Cluster:  cluster,
			Services: []string{*serviceArn},
		})
		if err != nil {
			log.Debug(err.Error())
			log.Fail("waiting for app to be removed failed")
			return
		}
		log.Info("updating app...")
		err = stack.Update(templateFileName, parameters)
	} else {
		log.Info("creating app...")
		err = stack.Create(templateFileName, parameters)
	}
	if err != nil {
		log.Debug(err.Error())
		log.Failf("fail to deploy %s", appName)
	}

	log.Succeed()
}

func (session *Session) deployApp(appName string, image string, envName string) {
	log := logger.NewTaskLogger()
	defer log.LogDone()
	config := session.GetComputeConfig()
	ecsClusterExportName := config.EcsClusterStackName

	log.Info("getting app details...")

	stackName := fmt.Sprintf("app-%s", appName)
	parameters := []cloudformation.Parameter{
		{
			ParameterKey:   aws.String("AppName"),
			ParameterValue: aws.String(appName),
		},
		{
			ParameterKey:   aws.String("Image"),
			ParameterValue: aws.String(image),
		},
		{
			ParameterKey:   aws.String("EcsClusterExportName"),
			ParameterValue: aws.String(ecsClusterExportName),
		},
		{
			ParameterKey:   aws.String("EnvironmentName"),
			ParameterValue: aws.String(envName),
		},
	}
	templateFileName := "app.yml"
	cfnClient := cloudformation.New(session.config)
	stack := &Stack{name: stackName, api: cfnClient}
	stackDescription, err := stack.Describe()
	if stackDescription != nil {
		log.Info("updating app...")
		err = stack.Update(templateFileName, parameters)
	} else {
		log.Info("creating app...")
		err = stack.Create(templateFileName, parameters)
	}
	if err != nil {
		log.Debug(err.Error())
		log.Failf("fail to deploy %s", appName)
	}

	log.Succeed()
}

// Deploy builds, publish image and deploy service
func (session *Session) Deploy(parameters *provideTypes.DeployParameters) {
	appType := parameters.AppType
	appName := parameters.AppName
	path := parameters.Path
	domainName := parameters.DomainName
	domainEmail := parameters.DomainEmail
	envName := parameters.EnvironmentName

	id := generateID()
	if appType == provideTypes.FrontProxy {
		appName = string(cmdTypes.KnownAppFrontProxy)
		frontProxyPath := getFrontProxyPath()
		tag := buildImage(frontProxyPath, appName, id, map[string]*string{
			"DOMAIN_NAME":  &domainName,
			"DOMAIN_EMAIL": &domainEmail,
			"ENV_NAME":     &envName,
		})
		repository := session.createRepository(appName)
		session.uploadImage(tag, repository, id)
		imageId := fmt.Sprintf("%s:%s", *repository, id)
		session.deployFrontProxy(imageId, envName)
		return
	}

	tag := buildImage(path, appName, id, map[string]*string{})
	repository := session.createRepository(appName)
	session.uploadImage(tag, repository, id)
	imageId := fmt.Sprintf("%s:%s", *repository, id)
	session.deployApp(appName, imageId, envName)
}
