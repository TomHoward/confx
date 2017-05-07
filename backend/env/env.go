package env

import (
	"errors"
	"fmt"
	"os"
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
	backend := EnvBackend{}

	for option, value := range options {
		switch option {
		case "ignoreUninitialised":
			if ignoreUninitialised, ok := value.(bool); ok {
				backend.ignoreUninitialised = ignoreUninitialised
			} else {
				return backend, errors.New(fmt.Sprintf("Invalid type for option '%s', expected bool", option))
			}
		}
	}

	return backend, nil
}
