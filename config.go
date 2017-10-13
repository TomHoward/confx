package main

import (
	"github.com/BurntSushi/toml"
	"github.com/tomhoward/confx/backend"
	"github.com/tomhoward/confx/backend/consul"
	"github.com/tomhoward/confx/backend/env"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

const DEFAULT_CONFIG_PATH = "/etc/confx/conf.d"

type config struct {
	Template configTemplate
	Sources  []configSource
}

func (c *config) GetValues() map[string]interface{} {
	values := map[string]interface{}{}

	for _, source := range c.Sources {
		var backend backend.Backend
		var err error

		keys := map[string]string{}
		// remove keys we've already collected
		for key, remoteKey := range source.Keys {
			if _, found := values[key]; !found {
				keys[key] = remoteKey
			}
		}

		if len(keys) == 0 {
			continue
		}

		switch source.Name {
		case "env":
			backend, err = env.New(source.Options)
		case "consul":
			backend, err = consul.New(source.Options)
		default:
			log.Fatalf("Unknown source type '%s'", source.Name)
		}

		if err != nil {
			log.Fatalf("%s: %s", source.Name, err)
		}

		sourceValues, err := backend.GetValues(keys)
		if err != nil {
			log.Fatalf("%s: %s\n", source.Name, err)
		}

		for k, v := range sourceValues {
			values[k] = v
		}
	}

	return values
}

type configSource struct {
	Name    string
	Keys    map[string]string
	Options map[string]interface{}
}

type configTemplate struct {
	Src         string
	Dest        string
	Gid         *int64
	Uid         *int64
	Permissions *string
}

type rawConfigTemplate struct {
	Src         string
	Dest        string
	Gid         int64
	Uid         int64
	Permissions string
}

type rawConfig struct {
	Template rawConfigTemplate
	Source   map[string]map[string]interface{}
}

func parseEnvVariableString(input string) string {
	envVariableRegex := regexp.MustCompile("\\$\\{[a-zA-Z0-9_]+}")

	matchIndexes := envVariableRegex.FindAllStringSubmatchIndex(input, -1)

	if matchIndexes == nil {
		return input
	}
	output := input

	for _, matchIndex := range matchIndexes {
		toReplace := input[matchIndex[0]:matchIndex[1]]
		// remove "${" and "}" from around the name
		envVariableName := toReplace[2 : len(toReplace)-1]

		output = strings.Replace(output, toReplace, os.Getenv(envVariableName), 1)
	}

	return output
}

func parseEnvVariablesInOptions(options map[string]interface{}) map[string]interface{} {
	output := map[string]interface{}{}

	for option, rv := range options {
		if value, ok := rv.(string); ok {
			output[option] = parseEnvVariableString(value)
		} else {
			output[option] = rv
		}
	}

	return output
}

func parseConfig(path string) (*config, error) {
	rawConfig := rawConfig{}
	config := config{}

	metadata, err := toml.DecodeFile(path, &rawConfig)
	if err != nil {
		return nil, err
	}

	template := configTemplate{}

	if rawConfig.Template.Src == "" {
		log.Fatalf("src not specified for template %s\n", path)
	} else {
		template.Src = parseEnvVariableString(rawConfig.Template.Src)
	}

	if rawConfig.Template.Dest == "" {
		log.Fatalf("dest not specified for template %s\n", path)
	} else {
		template.Dest = parseEnvVariableString(rawConfig.Template.Dest)
	}

	// use metadata to work out which template keys are set
	for _, k := range metadata.Keys() {
		switch k.String() {
		case "template.uid":
			template.Uid = &rawConfig.Template.Uid
		case "template.gid":
			template.Gid = &rawConfig.Template.Gid
		case "template.permissions":
			template.Permissions = &rawConfig.Template.Permissions
		}
	}

	config.Template = template

	sources := map[string]configSource{}

	for sourceName, data := range rawConfig.Source {
		source := configSource{
			Name: sourceName,
			Keys: map[string]string{},
		}
		for k, rv := range data {
			if k == "options" {
				if options, ok := rv.(map[string]interface{}); ok {
					source.Options = parseEnvVariablesInOptions(options)
				} else {
					log.Fatal("Invalid type for options")
				}
			} else {
				if value, ok := rv.(string); ok {
					source.Keys[k] = parseEnvVariableString(value)
				} else {
					log.Fatal("Invalid type for value")
				}
			}
		}

		sources[sourceName] = source
	}

	// use metadata to get cascading right
	for _, k := range metadata.Keys() {
		if strings.HasPrefix(k.String(), "source.") {
			splitKey := strings.SplitN(k.String(), ".", 3)

			// skip source options, subkeys etc
			if len(splitKey) > 2 {
				continue
			}

			sourceName := splitKey[1]

			if source, ok := sources[sourceName]; ok {
				config.Sources = append(config.Sources, source)
				delete(sources, sourceName)
			}

			if len(sources) == 0 {
				break
			}
		}
	}

	// reverse sources as we want to process bottom to top
	config.Sources = reverseSources(config.Sources)

	return &config, nil
}

func reverseSources(sources []configSource) []configSource {
	output := []configSource{}

	for i := len(sources) - 1; i >= 0; i-- {
		output = append(output, sources[i])
	}

	return output
}

func getConfigFilePaths(configDirPath string) ([]string, error) {
	filePaths := []string{}

	files, err := ioutil.ReadDir(configDirPath)

	if err != nil {
		return filePaths, err
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".toml") {
			filePaths = append(filePaths, path.Join(configDirPath, f.Name()))
		}
	}

	return filePaths, nil
}
