# Background

Getting really tiresome to experiment `stuff` with multiple scripts. This is an attempt for one cli to rule them all

Additionally it's is helping me learn how to write go command line interface using cobra (https://github.com/spf13/cobra)

# Prerequisites

## aws

- [awsweb](https://github.com/glassechidna/awsweb)

# How to add new commands

`cobra add <cmdname>`

# Notes

In order to use logger, make sure `startLog` is called first

# Useful scripts

remove images
```
docker images --filter "reference=<accountid>.dkr.ecr.ap-southeast-2.amazonaws.com/sample" --format "{{.ID}}"| %{ docker rmi $_ --force}
```

# Using

## Deploy front proxy  

```
    devenv deploy --type front-proxy
```

## Deploy apps

```
    devenv deploy appName --path <folder where Dockerfile is, exclude if current folder has it>
```

## Start environment

```
    devenv start
```

## Stop environment

```
    devenv stop
```

## Initialise environment

```
    devenv init
```

## Tear down environment

```
    devenv teardown
```

## Initialise configuration

```
    devenv init config
```

## Ssh into ec2 instance

```
    rm ~/.ssh/known_hosts
    ssh -i ~/.ssh/ecs-instance-ap-southeast-2.pem ec2-user@<domain>
``` 