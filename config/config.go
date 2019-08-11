package config

import (
	"encoding/json"
	"io/ioutil"
)

const (
	defaultPort              = "8080"
	defaultPasswdFilePath    = "/etc/passwd"
	defaultGroupFilePath     = "/etc/group"
	defaultWriteTimeoutInSec = 30
	defaultReadTimeoutInSec  = 30
	defaultIdleTimeoutInSec  = 60
)

// Config loads its fields from the configuration file that user provide, or uses the default settings
type Config struct {
	ListenHost        string
	Port              string
	WriteTimeoutInSec int
	ReadTimeoutInSec  int
	IdleTimeoutInSec  int
	RestDomain        string
	LogFilePath       string
	PasswdFilePath    string
	GroupFilePath     string
}

// Init loads the configuration file at configFilePath if len(configFilePath) > 0
func Init(configFilePath string) (setting *Config, err error) {
	setting = &Config{
		ListenHost:        "",
		Port:              defaultPort,
		WriteTimeoutInSec: defaultWriteTimeoutInSec,
		ReadTimeoutInSec:  defaultReadTimeoutInSec,
		IdleTimeoutInSec:  defaultIdleTimeoutInSec,
		RestDomain:        "",
		LogFilePath:       "",
		PasswdFilePath:    defaultPasswdFilePath,
		GroupFilePath:     defaultGroupFilePath,
	}

	if len(configFilePath) == 0 {
		return
	}

	b, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(b, setting); err != nil {
		return nil, err
	}
	return
}
