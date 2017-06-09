package env

import (
	"errors"
	"fmt"
	"os"
	strings "strings"
)

type EnvBackend struct {
	ignoreUninitialised bool
}

func (b EnvBackend) GetValues(keyPairs map[string]string) (map[string]string, error) {
	output := map[string]string{}

	for name, envVariableName := range keyPairs {
		value := os.Getenv(envVariableName)

		if b.ignoreUninitialised && value == "" {
			continue
		} else if value == "" {
			return output, errors.New(fmt.Sprintf("No value for '%s'", envVariableName))
		}

		output[name] = value
	}

	return output, nil
}

func New(options map[string]interface{}) (EnvBackend, error) {
	validOptions := []string{"ignore_uninitialised"}
	backend := EnvBackend{}

	for option, value := range options {
		switch option {
		case "ignore_uninitialised":
			if ignoreUninitialised, ok := value.(bool); ok {
				backend.ignoreUninitialised = ignoreUninitialised
			} else {
				return backend, errors.New(fmt.Sprintf("Invalid type for option '%s', expected bool", option))
			}
		default:
			return backend, errors.New(
				fmt.Sprintf("Unrecognised option '%s', possible options are %s",
					option,
					strings.Join(validOptions, ", "),
				))
		}
	}

	return backend, nil
}
