package service_config

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	log "github.com/sirupsen/logrus"
)

const DEFAULT_PARAM_DESCRIPTION = "Parameter set by golem"

type Identifier struct {
	Environment string
	Stack       string
	Service     string
}

type Configuration struct {
	Name  string
	Value string
}

func List(identifier Identifier) ([]Configuration, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return []Configuration{}, err
	}
	ssmClient := ssm.NewFromConfig(cfg)
	clusterPrefix := ssmKeyPrefix(identifier)
	params := &ssm.GetParametersByPathInput{
		Path:           aws.String(clusterPrefix),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}
	mapConfigs := make(map[string]Configuration)
	paginator := ssm.NewGetParametersByPathPaginator(ssmClient, params)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return []Configuration{}, err
		}
		for _, param := range page.Parameters {
			envName := parseEnvName(identifier, aws.ToString(param.Name))
			if envName == "" {
				continue
			}

			env, exists := mapConfigs[envName]
			if exists {
				// The fact is service keys are always longer than its corresponding cluster keys
				envKey := aws.ToString(param.Name)
				if len(envKey) > len(env.Name) {
					mapConfigs[envName] = Configuration{
						Name:  envKey,
						Value: aws.ToString(param.Value),
					}
				}
			} else {
				mapConfigs[envName] = Configuration{
					Name:  aws.ToString(param.Name),
					Value: aws.ToString(param.Value),
				}
			}
		}
	}

	configs := []Configuration{}
	for envName, config := range mapConfigs {
		configs = append(configs, Configuration{
			Name:  envName,
			Value: config.Value,
		})
	}
	return configs, nil
}

func SetEnv(identifier Identifier, configurations []Configuration) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Infof("Failed to load AWS config: %v", err)
		return
	}
	ssmClient := ssm.NewFromConfig(cfg)
	prefix := ssmKeyPrefix(identifier)
	if identifier.Service != "" {
		prefix = prefix + identifier.Service + "/"
	}

	application := identifier.Stack
	if identifier.Service != "" {
		application = identifier.Service
	}
	tags := []types.Tag{
		{
			Key:   aws.String("Environment"),
			Value: aws.String(identifier.Environment),
		},
		{
			Key:   aws.String("Owner"),
			Value: aws.String("golem"),
		},
		{
			Key:   aws.String("CreatedBy"),
			Value: aws.String("golem"),
		},
		{
			Key:   aws.String("Application"),
			Value: aws.String(application),
		},
		{
			Key:   aws.String("Stack"),
			Value: aws.String(identifier.Stack),
		},
	}
	for _, envVar := range configurations {
		result, error := ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
			Names: []string{
				prefix + envVar.Name,
			},
			WithDecryption: aws.Bool(true),
		})
		if error != nil {
			log.Infof("Failed to GetParameters %s:\n%+v\n", envVar.Name, error)
		}

		if result != nil && len(result.Parameters) > 0 {
			if envVar.Value != "" {
				_, error := ssmClient.PutParameter(context.TODO(), &ssm.PutParameterInput{
					Name:        aws.String(prefix + envVar.Name),
					Description: aws.String(DEFAULT_PARAM_DESCRIPTION),
					Value:       aws.String(envVar.Value),
					Type:        types.ParameterTypeString,
					Overwrite:   aws.Bool(true),
				})
				if error != nil {
					log.Infof("Failed to PutParameter %s:\n%+v\n", envVar.Name, error)
				} else {
					param := result.Parameters[0]
					_, error = ssmClient.AddTagsToResource(context.TODO(), &ssm.AddTagsToResourceInput{
						ResourceId:   param.ARN,
						ResourceType: types.ResourceTypeForTaggingParameter,
						Tags:         tags,
					})
					if error != nil {
						log.Infof("Failed to AddTagsToResource %s:\n%+v\n", envVar.Name, error)
					}
				}
			} else {
				_, error := ssmClient.DeleteParameter(context.TODO(), &ssm.DeleteParameterInput{
					Name: aws.String(prefix + envVar.Name),
				})
				if error != nil {
					log.Infof("Failed to DeleteParameter %s:\n%+v\n", envVar.Name, error)
				}
			}
		} else if envVar.Value != "" {
			_, error := ssmClient.PutParameter(context.TODO(), &ssm.PutParameterInput{
				Name:        aws.String(prefix + envVar.Name),
				Description: aws.String(DEFAULT_PARAM_DESCRIPTION),
				Value:       aws.String(envVar.Value),
				Type:        types.ParameterTypeString,
				Tags:        tags,
			})
			if error != nil {
				log.Infof("Failed to PutParameter %s:\n%+v\n", envVar.Name, error)
			}
		}
	}
}

func GetEnv(identifier Identifier, configNames []string) ([]Configuration, error) {
	paramKeys := []string{}
	prefix := ssmKeyPrefix(identifier)
	for _, paramName := range configNames {
		paramKeys = append(paramKeys, prefix+paramName)
		if identifier.Service != "" {
			paramKeys = append(paramKeys, prefix+identifier.Service+"/"+paramName)
		}
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return []Configuration{}, err
	}
	ssmClient := ssm.NewFromConfig(cfg)
	mapConfigs := make(map[string]Configuration)
	for i := 0; i < len(paramKeys); i += 10 {
		tail := i + 10
		if tail > len(paramKeys) {
			tail = len(paramKeys)
		}
		chunks := paramKeys[i:tail]

		params, error := ssmClient.GetParameters(context.TODO(), &ssm.GetParametersInput{
			Names:          chunks,
			WithDecryption: aws.Bool(true),
		})
		if error != nil {
			return []Configuration{}, error
		}
		for _, param := range params.Parameters {
			envName := parseEnvName(identifier, aws.ToString(param.Name))
			if envName == "" {
				continue
			}

			env, exists := mapConfigs[envName]
			if exists {
				// The fact is service keys are always longer than its corresponding cluster keys
				envKey := aws.ToString(param.Name)
				if len(envKey) > len(env.Name) {
					mapConfigs[envName] = Configuration{
						Name:  envKey,
						Value: aws.ToString(param.Value),
					}
				}
			} else {
				mapConfigs[envName] = Configuration{
					Name:  aws.ToString(param.Name),
					Value: aws.ToString(param.Value),
				}
			}
		}
	}
	configs := []Configuration{}
	for envName, config := range mapConfigs {
		configs = append(configs, Configuration{
			Name:  envName,
			Value: config.Value,
		})
	}
	return configs, nil
}

func ParseEnvString(envString string) (Configuration, error) {
	parts := strings.SplitN(envString, "=", 2)
	if len(parts) == 1 {
		return Configuration{
			Name:  parts[0],
			Value: "",
		}, nil
	} else if len(parts) == 2 {
		return Configuration{
			Name:  parts[0],
			Value: parts[1],
		}, nil
	}

	return Configuration{}, fmt.Errorf("Unable to parse environment variable string")
}

func ssmKeyPrefix(id Identifier) string {
	return "/golem/" + id.Environment + "/" + id.Stack + "/"
}
func parseEnvName(identifier Identifier, paramKey string) string {
	if !strings.HasPrefix(paramKey, ssmKeyPrefix(identifier)) {
		return ""
	}

	parts := strings.SplitN(paramKey, "/", 5)
	if len(parts) == 5 {
		if identifier.Service == "" || identifier.Service != parts[3] {
			return ""
		}
	} else if len(parts) != 4 {
		return ""
	}

	return parts[len(parts)-1]
}
