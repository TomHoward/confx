package consul

import (
	"errors"
	"fmt"
	consulapi "github.com/hashicorp/consul/api"
	"strings"
)

type ConsulBackend struct {
	address   string // host:port
	token     string // acl token
	useSSL    *bool
	verifySSL *bool
}

func (b ConsulBackend) GetValues(keyPairs map[string]string) (map[string]interface{}, error) {
	output := map[string]interface{}{}

	config := consulapi.DefaultConfig()
	if b.address != "" {
		config.Address = b.address
	}
	if b.useSSL != nil {
		if *b.useSSL {
			config.Scheme = "https"
		} else {
			config.Scheme = "http"
		}
	}
	if config.Scheme == "https" && b.verifySSL != nil {
		config.TLSConfig.InsecureSkipVerify = *b.verifySSL
	}

	consul, err := consulapi.NewClient(config)
	if err != nil {
		return output, err
	}

	kv := consul.KV()

	var kvQueryOptions *consulapi.QueryOptions
	if b.token != "" {
		kvQueryOptions = &consulapi.QueryOptions{
			Token: b.token,
		}
	}

	for name, consulKey := range keyPairs {
		kvPair, _, err := kv.Get(consulKey, kvQueryOptions)
		if err != nil {
			return output, err
		}

		if kvPair == nil {
			return output, errors.New(fmt.Sprintf("No value for '%s', have you specified the necessary token?", consulKey))
		}

		output[name] = string(kvPair.Value)
	}

	return output, nil
}

func New(options map[string]interface{}) (ConsulBackend, error) {
	validOptions := []string{"address", "ssl", "verify_ssl"}
	backend := ConsulBackend{}

	// TODO: basic auth support
	// TODO: support multiple addresses
	// TODO: remove repeated logic below

	for option, value := range options {
		switch option {
		case "token":
			if token, ok := value.(string); ok {
				backend.token = token
			} else {
				return backend, errors.New(fmt.Sprintf("Invalid type for option '%s', expected string", option))
			}
		case "address":
			if address, ok := value.(string); ok {
				// TODO: regex to validate address is in format host:port
				backend.address = address
			} else {
				return backend, errors.New(fmt.Sprintf("Invalid type for option '%s', expected string", option))
			}
		case "ssl":
			if ssl, ok := value.(bool); ok {
				backend.useSSL = &ssl
			} else {
				return backend, errors.New(fmt.Sprintf("Invalid type for option '%s', expected bool", option))
			}
		case "verify_ssl":
			if verifySSL, ok := value.(bool); ok {
				backend.verifySSL = &verifySSL
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
