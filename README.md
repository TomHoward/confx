# Confx

Confx is a minimal configuration management tool which makes it easy to inject values from different backend sources. It is inspired by [confd](https://github.com/kelseyhightower/confd) but has a few key differences:-
 
 * Has no concept of watching backends nor process management
 * Allows the use of multiple backends at the same time
 * Allows value overriding by cascading backends
 * Allows the use of environment variables in its own configurations
 
 #### Supported backends
 * Environment variables
 * Consul
 
 ### Usage
 ```shell
 # by default confx expects configurations in /etc/confx/conf.d and templates in /etc/confx/templates
 confx
 
 # you may optionally provide a configuration or template directory to use
 confx -c /home/tom/confx/conf.d -t /home/tom/confx/templates
```
 
 #### Configuration
 ```toml
 # Configurations are defined in TOML files
 
 # Every configuration must define a template with a source relative to the configuration
 # directory provided when launching and a destination
 [template]
 src = "some-settings.conf.tmpl"
 dest = "/etc/some-program/some-settings.conf"
 # Optionally confx can also set the permissions or uid/gid for the output file
 uid = 123
 gid = 456
 permissions = "600"
 
 # backends are defined as sources with its subkeys and values being template relative keys and lookup keys for the source
 [source.consul]
 VALUE_IN_TEMPLATE = "path/to/value/in/consul"
 DATABASE_HOST = "myapp/settings/database_host"
 # it is possible to use environment variables within configuration files, here 
 # ${COUNTRY} will be replaced with the value of the COUNTRY environment variable
 COUNTRY_SPECIFIC_VAR = "myapp/settings/${COUNTRY}/some_setting"
 
 # You can optionally also provide parameters to the source to config it
 [source.consul.options]
 # (optional) address of consul node, in format host:port
 address = "myconsul.host:6789"
 # (optional) should SSL be used for the connection (default false)
 ssl = true
 # (optional) should SSL certificates be verified (default true)
 verify_ssl = false
 
 # backends cascade in a similar way to CSS, this allows us to override values, e.g. for use in a local development environment
 # here DATABASE_HOST will be override the value from consul if the environment variable is set
 [source.env]
 VALUE_IN_TEMPLATE = "ENVIRONMENT_VARIABLE_NAME"
 DATABASE_HOST = "DATABASE_HOST"
 DATABASE_PASSWORD = "DATABASE_PASSWORD"
 
 [source.env.options]
 # (optional) don't throw an error if an expect environment variable isn't set, instead we can use default values in the template
 ignore_uninitialised = true
 ```
 
 ####  Templates
 Templates are standard `go/text` templates, use the function `getV` to access a value provided by a backend. You can optionally pass a second parameter to `getV` to specify a default value if the backend hasn't returned one.
 ```yaml
 ---
host: 0.0.0.0
port: {{getV DATABASE_PORT}}
database_name: {{getV DATABASE_NAME "somedatabase"}}
database_password: {{getV DATABASE_PASSWORD}}
```

There are also other helper functions available
* `getEnv "key" "default-value"` - (string) reads a value from an environment variable (independent of it being specified as a backend), takes an optional default value.
* `hasV "key"` - (bool)
* `hasEnv "key"` - (bool)
* `hasPrefix "string" "prefix"` - (bool)
* `hasSuffix "string" "prefix"` - (bool)
* `contains "search-in" "search-for"` - (bool)
* `toUpper "lower-case-string"` - (string)
* `toLower "UPPER-CASE-STRING"` - (string)
* `split "split,me,up" ","` - ([]string)