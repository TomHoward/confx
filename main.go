package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"
)

const DEFAULT_TEMPLATE_PATH = "/etc/confx/templates"

// confx version and commit ID, set during compilation
// go build -ldflags "-X main.version=0.0.2 -X main.commitId=$(git rev-parse --short HEAD)"
var version string
var commitId string

func randomStr(n int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	var chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
	output := make([]rune, n)
	for i := range output {
		output[i] = chars[rand.Intn(len(chars))]
	}
	return string(output)
}

func main() {
	configurationDirFlag := flag.String("c", "", "configuration path directory")
	templatesDirFlag := flag.String("t", "", "template path directory")
	versionFlag := flag.Bool("v", false, "print version number")

	flag.Parse()

	if versionFlag != nil && *versionFlag {
		fmt.Printf("confx v%s (%s) | https://github.com/tomhoward/confx\n", version, commitId)
		os.Exit(0)
	}

	var templatesPathDir string
	var configPathDir string

	if configurationDirFlag == nil || *configurationDirFlag == "" {
		configPathDir = DEFAULT_CONFIG_PATH
	} else {
		configPathDir = *configurationDirFlag
	}

	if templatesDirFlag == nil || *templatesDirFlag == "" {
		templatesPathDir = DEFAULT_TEMPLATE_PATH
	} else {
		templatesPathDir = *templatesDirFlag
	}

	if _, err := os.Stat(configPathDir); err != nil {
		log.Fatalf("Could not find config dir: %s", configPathDir)
	}

	if _, err := os.Stat(templatesPathDir); err != nil {
		log.Fatalf("Could not find templates dir: %s", templatesPathDir)
	}

	configPaths, err := getConfigFilePaths(configPathDir)

	if err != nil {
		log.Fatalf("Could not access config dir: %s", configPathDir)
	}

	if len(configPaths) == 0 {
		log.Fatalf("No config files found in %s\n", configPathDir)
	}

	for _, configPath := range configPaths {
		config, err := parseConfig(configPath)
		if err != nil {
			log.Fatalf("Could not parse config for %s: %s\n", configPath, err)
		}

		templateSrc := path.Join(templatesPathDir, config.Template.Src)

		log.Printf("%s -> %s", templateSrc, config.Template.Dest)

		values := config.GetValues()
		tmpl, err := template.New("main").Funcs(template.FuncMap{
			"getV": func(key string, defaultValue ...string) interface{} {
				if value, ok := values[key]; ok {
					return value
				} else if len(defaultValue) >= 1 {
					return defaultValue[0]
				} else {
					log.Fatalf("Error: No value specified for \"%s\"\n", key)
					return ""
				}
			},
			"hasV": func(key string) bool {
				_, ok := values[key]
				return ok
			},
			"getEnv": func(key string, defaultValue ...string) string {
				value := os.Getenv(key)

				if value == "" && len(defaultValue) > 0 {
					return defaultValue[0]
				} else if len(value) > 0 {
					return value
				} else {
					log.Fatalf("Error: No value specified for env variable \"%s\"\n", key)
					return ""
				}
			},
			"hasEnv": func(key string) bool {
				value := os.Getenv(key)
				return value != ""
			},
			"hasPrefix": func(s string, prefix string) bool {
				return strings.HasPrefix(s, prefix)
			},
			"hasSuffix": func(s string, prefix string) bool {
				return strings.HasPrefix(s, prefix)
			},
			"contains": func(s string, needle string) bool {
				return strings.Contains(s, needle)
			},
			"toUpper": func(s string) string {
				return strings.ToUpper(s)
			},
			"toLower": func(s string) string {
				return strings.ToLower(s)
			},
			"split": func(s string, delimiter string) []string {
				return strings.Split(s, delimiter)
			},
		}).ParseFiles(templateSrc)

		if err != nil {
			log.Fatalf("Could not parse template: %s\n", err)
		}

		var permissions os.FileMode
		if config.Template.Permissions != nil {
			intPermissions, err := strconv.ParseUint(*config.Template.Permissions, 8, 64)
			if err != nil {
				log.Fatal("Invalid value for permissions")
			}
			permissions = os.FileMode(intPermissions)
		} else {
			permissions = os.FileMode(0644)
		}

		// open output file in temp dir in case there are errors writing the template, we don't want to
		// overwrite the file with nothing
		outputFilePath := path.Join(os.TempDir(), randomStr(16))
		outputFile, err := os.OpenFile(outputFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permissions)
		if err != nil {
			log.Fatal(outputFile)
		}

		if err := tmpl.ExecuteTemplate(outputFile, config.Template.Src, map[string]string{}); err != nil {
			log.Fatal(err)
		}

		if err := outputFile.Close(); err != nil {
			log.Fatal(err)
		}

		// move from temp location to actual location
		if err := os.Rename(outputFilePath, config.Template.Dest); err != nil {
			log.Fatal(err)
		}

		// set uid and gid
		if config.Template.Gid != nil || config.Template.Uid != nil {
			currentUser, err := user.Current()

			var uid int64
			var gid int64

			if config.Template.Uid == nil {
				if err != nil {
					log.Fatal("Could not set uid, error getting current user")
				}
				uid, err = strconv.ParseInt(currentUser.Uid, 10, 64)
				if err != nil {
					panic("Unable to parse uid for current user")
				}
			} else {
				uid = *config.Template.Uid
			}

			if config.Template.Gid == nil {
				if err != nil {
					log.Fatal("Could not set gid, error getting current user")
				}
				gid, err = strconv.ParseInt(currentUser.Gid, 10, 64)
				if err != nil {
					panic("Unable to parse gid for current user")
				}
			} else {
				gid = *config.Template.Gid
			}

			err = syscall.Chown(config.Template.Dest, int(uid), int(gid))
			if err != nil {
				log.Fatal(fmt.Sprintf("Error: Could not set uid/gid: %s", err))
			}
		}
	}
}
