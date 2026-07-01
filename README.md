# Golem service

Manage services run on AWS

# Require arguments

Golem service commands perform on an specific service on an environment. That's why argument `-e`/`--environment` and `-t`/`--stack` are required. This is the same convention from `golem-tf`.<br/>
Further, `-s`/`--service` is required for commands need to perform on specific service in stack. 

# Install

For now, the only way to install `golem-service` is from source code. Pull and run `make build`. Then run from repo root directory.

# Subcommands

## config

Get/set configuartions for service. In AWS ECS platform, configurations are set through Parameter Store or AWS Secret. 


```shell
$ golem-service config
$ golem-service config list
$ golem-service config set NAME=value NAME2="value 2"
$ golem-service config get NAME NAME2
```

## ps

List and manage processes in services

```shell
$ golem-service ps
$ golem-service ps kill
```

## deploy

Deploy and update AWS ECS service tasks.

```shell
$ golem-deploy <cluster> <service> [options]
```

