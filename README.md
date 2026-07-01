# 🗿 Golem

`golem` is set of personal tools I used to operate AWS infrastructure, AWS ECS specificly. 
It was as a gap filler when I was moving from Heroku to AWS ECS. 
It looks like revolving around https://12factor.net/ and heroku cli. But much more bias on my personal experience and workflow, so I recommend of using it with caution.

# Features

- Manage ECS task's environment variable, secret through AWS Parameter store, AWS Secrets
- List tasks in service
- Deploy to AWS ECS
- Entrypoint which support SSH, recycling...
- Connect to task. Yes, the controlversal `heroku console` `heroku rails c` 😆
- ...

# Install

Download from repository releases. 
I split subcommands to different binaries to use in specific situations, but still keep the master binary which run all the subcommands.

# Usage

## config

Get/set configurations for service. In AWS ECS platform, configurations are set through Parameter Store or AWS Secret. 


```shell
$ golem-config
$ golem-config list
$ golem-config set NAME=value NAME2="value 2"
$ golem-config get NAME NAME2
```

## ps

List and manage processes in services

```shell
$ golem-ps
$ golem-ps kill
```

## deploy

Deploy and update AWS ECS service tasks.

```shell
$ golem-deploy <cluster> <service> [options]
```

# Examples

TBA

# Licencse

TBA. Maybe MIT - anyone could use for anything, just don't sue me 😝

