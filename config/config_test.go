package config

import (
	"testing"
)

const (
	testConfigFile = "../testData/config.json"

	dummyListenHost     = "127.0.0.1"
	dummyPort           = "4321"
	dummyTimeoutInSec   = 4321
	dummyRestDomain     = "127.0.0.1"
	dummyLogFile        = "./testData/log"
	dummyPasswdFilePath = "./testData/passwd"
	dummyGroupFilePath  = "./testData/group"
)

func assert(t *testing.T, condition bool) {
	if !condition {
		t.Fatal()
	}
}

func TestConfigLoadFromFile(t *testing.T) {
	setting, err := Init(testConfigFile)
	assert(t, err == nil)

	assert(t, setting.ListenHost == dummyListenHost)
	assert(t, setting.Port == dummyPort)
	assert(t, setting.IdleTimeoutInSec == dummyTimeoutInSec)
	assert(t, setting.WriteTimeoutInSec == dummyTimeoutInSec)
	assert(t, setting.ReadTimeoutInSec == dummyTimeoutInSec)
	assert(t, setting.RestDomain == dummyRestDomain)
	assert(t, setting.LogFilePath == dummyLogFile)
	assert(t, setting.PasswdFilePath == dummyPasswdFilePath)
	assert(t, setting.GroupFilePath == dummyGroupFilePath)
}
