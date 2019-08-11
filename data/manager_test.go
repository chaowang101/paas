package data

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

const (
	originalPasswdPath = "../testData/passwd"
	originalGroupPath  = "../testData/group"

	passwdPath = "../testData/passwd_cp"
	groupPath  = "../testData/group_cp"

	passwdPathRenamed = "../testData/passwd_cp2"
)

var mgr Manager

func setup() {
	_ = os.Remove(groupPath)
	_ = os.Remove(passwdPath)

	cmd := exec.Command("cp", originalGroupPath, groupPath)
	err := cmd.Run()
	if err != nil {
		panic("Fail to copy the test data of group file, err: " + err.Error())
	}

	cmd = exec.Command("cp", originalPasswdPath, passwdPath)
	err = cmd.Run()
	if err != nil {
		panic("Fail to copy the test data of passwd file, err: " + err.Error())
	}

	if mgr, err = NewManager(passwdPath, groupPath); err != nil {
		panic("Fail to create manager, err: " + err.Error())
	}

	err = mgr.Start()
	if err != nil {
		panic("Fail to start the manager, err: " + err.Error())
	}

}

func teardown() {
	mgr.Stop()
	os.Remove(passwdPath)
	os.Remove(groupPath)
}

func runTests(m *testing.M) int {
	setup()
	defer teardown()
	return m.Run()
}

func assert(t *testing.T, condition bool) {
	if !condition {
		t.Fatal()
	}
}

func TestMain(m *testing.M) {
	code := runTests(m)
	os.Exit(code)
}

func TestGetUsers(t *testing.T) {
	user := mgr.GetUserByUID("0")
	assert(t, user != nil)
	assert(t, user.Name == "root")
	assert(t, user.UID == "0")
	assert(t, user.GID == "0")
	assert(t, user.Comment == "System Administrator")
	assert(t, user.Home == "/var/root")
	assert(t, user.Shell == "/bin/sh")

	user = mgr.GetUserByUID("999")
	assert(t, user == nil)

	res := mgr.GetUserByQuery("daemon", "1", "1", "System Services", "/var/root", "/usr/bin/false")
	assert(t, len(res) == 1)

	res = mgr.GetUserByQuery("", "", "", "", "", "/usr/bin/false")
	assert(t, len(res) == 4)

	res = mgr.GetUserByQuery("daemon", "1", "1", "System Services", "/var/root", "/usr/bin/true")
	assert(t, len(res) == 0)

	res = mgr.GetAllUsers()
	assert(t, len(res) == 6)
}

func TestGetGroups(t *testing.T) {
	group := mgr.GetGroupByGID("1")
	assert(t, group != nil)
	assert(t, group.Name == "daemon")
	assert(t, len(group.Members) == 1)
	assert(t, group.Members[0] == "root")

	group = mgr.GetGroupByGID("1000")
	assert(t, group == nil)

	var rootGroup [][]*Group
	rootGroup = append(rootGroup, mgr.GetGroupsByUID("0"))
	rootGroup = append(rootGroup, mgr.GetGroupByQuery("", "", []string{"root"}))
	for _, res := range rootGroup {
		assert(t, len(res) == 3)
		resGroupNameMap := map[string]struct{}{}
		for _, m := range res {
			resGroupNameMap[m.Name] = struct{}{}
		}
		for _, name := range []string{"daemon", "staff", "certusers"} {
			_, ok := resGroupNameMap[name]
			assert(t, ok)
		}
	}

	res := mgr.GetGroupsByUID("1000")
	assert(t, len(res) == 0)

	res = mgr.GetGroupByQuery("", "", []string{"root2"})
	assert(t, len(res) == 0)

	res = mgr.GetGroupByQuery("daemon", "1", []string{"root"})
	assert(t, len(res) == 1)

	res = mgr.GetGroupByQuery("daemon", "10", []string{"root"})
	assert(t, len(res) == 0)

	res = mgr.GetAllGroups()
	assert(t, len(res) == 8)
}

func testMonitorFile(t *testing.T) {
	f, err := os.OpenFile(passwdPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	assert(t, err == nil)
	defer f.Close()

	// test adding entries
	_, err = f.WriteString("root2:*:99:99:System Administrator:/var/root:/bin/sh\n")
	assert(t, err == nil)

	err = f.Sync()
	assert(t, err == nil)

	time.Sleep(1 * time.Second)
	user := mgr.GetUserByUID("99")
	assert(t, user != nil)

	// test deleting entries
	err = f.Truncate(0)
	assert(t, err == nil)

	err = f.Sync()
	assert(t, err == nil)

	time.Sleep(1 * time.Second)
	user = mgr.GetUserByUID("99")
	assert(t, user == nil)
}

func TestMonitorFile(t *testing.T) {
	user := mgr.GetUserByUID("99")
	assert(t, user == nil)
	// test the file is still watched after rename
	err := os.Rename(passwdPath, passwdPathRenamed)
	assert(t, err == nil)
	err = os.Rename(passwdPathRenamed, passwdPath)
	assert(t, err == nil)
	time.Sleep(2 * time.Second)
	testMonitorFile(t)

	// test the file is still watched after deletion and creation
	err = os.Remove(passwdPath)
	assert(t, err == nil)

	cmd := exec.Command("cp", originalPasswdPath, passwdPath)
	err = cmd.Run()
	if err != nil {
		panic("Fail to copy the test data of passwd file, err: " + err.Error())
	}
	time.Sleep(2 * time.Second)
	testMonitorFile(t)
}
