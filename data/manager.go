package data

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
)

// passwd file offset
const (
	userNameOffset = 0
	// skip password offset
	uidOffset = iota + 1
	userGidOffset
	commentOffset
	homeOffset
	shellOffset
)

// group file offset
const (
	groupNameOffset = 0
	// skip password offset
	groupGidOffset = iota + 1
	memberOffset
)

const (
	fieldDelim  = ":"
	memberDelim = ","

	numberOfFieldGroupEntry  = 4
	numberOfFieldPasswdField = 7
	// There is no async API to check for file creation. A loop with sleep time of 1 second
	// will be used to watch for file creation once the monitored has been deleted or renamed
	monitorFileCreationIntervalInSec = 1 * time.Second
)

// User is the data structure for each entry read from the /etc/passwd file
type User struct {
	Name    string `json:"name"`
	UID     string `json:"uid"`
	GID     string `json:"gid"`
	Comment string `json:"comment"`
	Home    string `json:"home"`
	Shell   string `json:"shell"`
}

// Group is the data structure for each entry read from the /etc/group file
type Group struct {
	Name    string   `json:"name"`
	GID     string   `json:"gid"`
	Members []string `json:"members"`

	memberSet map[string]struct{}
}

// Manager is used to retrieve the User or Group data structure
// In order for Manager to monitor the changes of the underlying files Start() must be call.
// And Stop() should be called for a graceful shutdown
type Manager interface {
	// Start enable the Manager to monitor the passwd and group files for any change
	Start() error
	// Stop will stop  Manager from monitoring the passwd and group files and free up resources
	Stop()

	// GetAllUsers returns all the users in the passwd file
	GetAllUsers() []*User
	// GetUserByQuery returns all the users that matching all of the specified query fields.
	// Only exact matches is supported. 204 SuccessNoContent will be returned if no data is found.
	GetUserByQuery(name, uid, gid, comment, home, shell string) []*User
	// GetUserByUID returns the user with UID, assuming there will be no duplicated UID
	// 404 will be returned if no group is found
	GetUserByUID(uid string) *User
	// GetAllGroups returns all the groups in the group file.
	// 204 SuccessNoContent will be returned if no data is found.
	GetAllGroups() []*Group
	// GetGroupsByUID all the groups for a given user with UID.
	// 204 SuccessNoContent will be returned if no data is found.
	GetGroupsByUID(uid string) []*Group
	// GetGroupByQuery returns all the groups matching all of the specified query fields.
	// Any group containing all the specified members wil be returned, i.e. when query members are a subset of
	// group members.  204 SuccessNoContent will be returned if no data is found.
	GetGroupByQuery(name, gid string, members []string) []*Group
	// GetGroupByGID returns the group with GID. Assuming GID is unique
	// 404 will be returned if no group is found
	GetGroupByGID(gid string) *Group
}

// Index User by UID and user name. This struct is immutable after construction
type userData struct {
	userMapByID   map[string]*User
	userMapByName map[string][]*User
	userSlice     []*User
}

// index Group by GID and group name. // This struct is immutable after construction
type groupData struct {
	groupMapByID   map[string]*Group
	groupMapByName map[string][]*Group
	groupSlice     []*Group
}

type manager struct {
	passwdFilePath string
	groupFilePath  string

	exit chan struct{}

	// using a reader/writer lock that favorite writer, assuming write operation is rare compare to read
	userLock  sync.RWMutex
	user      *userData
	groupLock sync.RWMutex
	group     *groupData
}

type handleFileUpdateFunc func(mgr *manager)

func (m *manager) GetAllUsers() []*User {
	m.userLock.RLock()
	defer m.userLock.RUnlock()

	return m.user.userSlice
}

func compareUser(gid, comment, home, shell string, user *User) bool {
	if len(gid) > 0 && user.GID != gid {
		return false
	}

	if len(comment) > 0 && user.Comment != comment {
		return false
	}

	if len(home) > 0 && user.Home != home {
		return false
	}

	if len(shell) > 0 && user.Shell != shell {
		return false
	}
	return true
}

func (m *manager) GetUserByQuery(name, uid, gid, comment, home, shell string) []*User {
	m.userLock.RLock()
	defer m.userLock.RUnlock()

	var res []*User

	// since uid is unique, it is guaranteed at most one user will match
	if len(uid) != 0 {
		user := m.GetUserByUID(uid)
		if user == nil {
			return res
		}
		if len(name) > 0 && user.Name != name {
			return res
		}
		if !compareUser(gid, comment, home, shell, user) {
			return res
		}
		res = append(res, user)
		return res
	}

	var candidate []*User
	if len(name) != 0 {
		if candidate = m.user.userMapByName[name]; len(candidate) == 0 {
			return res
		}
	} else {
		candidate = m.user.userSlice
	}

	for _, u := range candidate {
		if !compareUser(gid, comment, home, shell, u) {
			continue
		}
		res = append(res, u)
	}
	return res
}

func (m *manager) GetUserByUID(uid string) *User {
	m.userLock.RLock()
	defer m.userLock.RUnlock()
	return m.user.userMapByID[uid]
}

func (m *manager) GetGroupsByUID(uid string) []*Group {
	var res []*Group

	user := m.GetUserByUID(uid)
	if user == nil {
		return res
	}

	m.groupLock.RLock()
	for _, g := range m.group.groupSlice {
		if _, ok := g.memberSet[user.Name]; ok {
			res = append(res, g)
		}
	}
	m.groupLock.RUnlock()
	return res
}

func (m *manager) GetAllGroups() []*Group {
	m.groupLock.RLock()
	defer m.groupLock.RUnlock()

	return m.group.groupSlice
}

func (m *manager) GetGroupByQuery(name, gid string, members []string) []*Group {
	m.groupLock.RLock()
	defer m.groupLock.RUnlock()

	var res []*Group
	var candidate []*Group

	// As GID is unique, there could be at most one Group match a provided GID
	if len(gid) > 0 {
		group := m.group.groupMapByID[gid]
		if group == nil {
			return res
		}

		if len(name) > 0 && group.Name != name {
			return res
		}

		for _, m := range members {
			if _, ok := group.memberSet[m]; !ok {
				return res
			}
		}
		res = append(res, group)
		return res
	}

	if len(name) != 0 {
		if candidate = m.group.groupMapByName[name]; len(candidate) == 0 {
			return res
		}
	} else {
		candidate = m.group.groupSlice
	}

Loop:
	for _, g := range candidate {
		for _, memberInQuery := range members {
			if len(g.Members) == 0 {
				continue Loop
			}
			// (len(g.Members) > 0) guarantees (g.memberSet != nil)
			if _, ok := g.memberSet[memberInQuery]; !ok {
				continue Loop
			}
		}
		res = append(res, g)
	}
	return res
}

func (m *manager) GetGroupByGID(gid string) *Group {
	m.groupLock.RLock()
	defer m.groupLock.RUnlock()

	return m.group.groupMapByID[gid]
}

// In case the monitored file is deleted or renamed, it will keep watching for the
// monitored file to be recreated.
func (m *manager) waitForFileCreation(watcher *fsnotify.Watcher, path string) {
	for {
		// Exit if programming is terminating
		select {
		case <-m.exit:
			return
		default:
		}

		if exist, err := pathExists(path); !exist {
			if err != nil {
				log.Printf("Test path %s exist, err %s\n", path, err)
			}
			time.Sleep(monitorFileCreationIntervalInSec)
			continue
		}
		err := watcher.Watch(path)
		if err != nil {
			log.Printf("Fail to watch path %s, err %s\n", path, err)
		}
		break
	}
}

func (m *manager) watchFile(watcher *fsnotify.Watcher, path string, handler handleFileUpdateFunc) {
	log.Println("Start monitoring file ", path)
ForLoop:
	for {
		// exit channel has higher priority
		select {
		case <-m.exit:
			break ForLoop
		default:
		}

		select {
		case ev := <-watcher.Event:
			log.Println("file change event:", ev)
			if ev.IsDelete() {
				log.Printf("File %s is deleted\n", ev.Name)
				m.waitForFileCreation(watcher, path)
				continue ForLoop
			}
			if ev.IsRename() {
				log.Printf("File %s is moved\n", ev.Name)
				m.waitForFileCreation(watcher, path)
				continue ForLoop
			}

			if ev.IsModify() {
				handler(m)
			}

			// ev.IsAttrib() is ignored and ev.IsCreate() only applies to directory
		case err := <-watcher.Error:
			log.Printf("error %s for watching file %s\n", err, path)
			break ForLoop
		case <-m.exit:
			break ForLoop
		}
	}
	log.Println("Stop monitoring file ", path)
}

func handlePasswdFileUpdate(mgr *manager) {
	userDataObj, err := parsePasswdFile(mgr.passwdFilePath)
	if err != nil {
		log.Printf("Fail to update passwd file change due to error: %s\n", err)
		return
	}

	mgr.userLock.Lock()
	defer mgr.userLock.Unlock()
	mgr.user = userDataObj
}

func handleGroupFileUpdate(mgr *manager) {
	groupDataObj, err := parseGroupFile(mgr.groupFilePath)
	if err != nil {
		log.Printf("Fail to update group file change due to error: %s\n", err)
	}

	mgr.groupLock.Lock()
	defer mgr.groupLock.Unlock()
	mgr.group = groupDataObj
}

func (m *manager) Start() error {
	watcherPasswd, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcherPasswd.Watch(m.passwdFilePath)
	if err != nil {
		return err
	}

	watcherGroup, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcherGroup.Watch(m.groupFilePath)
	if err != nil {
		return err
	}

	go m.watchFile(watcherPasswd, m.passwdFilePath, handlePasswdFileUpdate)
	go m.watchFile(watcherGroup, m.groupFilePath, handleGroupFileUpdate)

	return nil

}

func (m *manager) Stop() {
	log.Println("Stopping password manager")
	close(m.exit)
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// NewManager instantiate a new data.Manager to retrieve data from provided passwd file and group file, it
// also monitor any change that happens to those files and update the content accordingly
func NewManager(passwdPath, groupPath string) (Manager, error) {
	for _, path := range []string{passwdPath, groupPath} {
		if res, err := pathExists(path); !res {
			if err != nil {
				log.Printf("Fail to find path %s, error:%s", path, err)
			}
			return nil, fmt.Errorf("File %s does not exist", path)
		}
	}

	managerObj := &manager{
		passwdFilePath: passwdPath,
		groupFilePath:  groupPath,
		exit:           make(chan struct{}),
	}

	//  managerObj.passwdFile and managerObj.groupFile will be closed in the watchFile()
	var err error
	managerObj.user, err = parsePasswdFile(managerObj.passwdFilePath)
	if err != nil {
		return nil, err
	}

	managerObj.group, err = parseGroupFile(managerObj.groupFilePath)
	if err != nil {
		return nil, err
	}

	return managerObj, nil
}

func parseGroup(line string) (*Group, error) {
	strList := strings.Split(strings.TrimSpace(line), fieldDelim)
	if len(strList) != numberOfFieldGroupEntry {
		return nil, fmt.Errorf("Malformed content %s", line)
	}
	res := &Group{
		Name:      strings.TrimSpace(strList[groupNameOffset]),
		GID:       strings.TrimSpace(strList[groupGidOffset]),
		Members:   make([]string, 0),
		memberSet: make(map[string]struct{}),
	}

	members := strList[memberOffset]
	if len(members) != 0 {
		memberList := strings.Split(strings.TrimSpace(members), memberDelim)
		res.Members = append(res.Members, memberList...)
		for _, m := range res.Members {
			res.memberSet[m] = struct{}{}
		}
	}

	return res, nil
}

func parseGroupFile(groupFilePath string) (*groupData, error) {
	buf, err := ioutil.ReadFile(groupFilePath)
	if err != nil {
		return nil, err
	}

	groupDataObj := &groupData{
		groupMapByID:   make(map[string]*Group),
		groupMapByName: make(map[string][]*Group),
		groupSlice:     make([]*Group, 0),
	}

	lines := strings.Split(strings.TrimSpace(string(buf)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// skip commented or empty lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		// parse the current line
		group, err := parseGroup(line)
		if err != nil {
			return nil, err
		}
		groupDataObj.groupSlice = append(groupDataObj.groupSlice, group)
		groupDataObj.groupMapByID[group.GID] = group
		groupDataObj.groupMapByName[group.Name] = append(groupDataObj.groupMapByName[group.Name], group)
	}
	return groupDataObj, nil
}

func parseUser(line string) (*User, error) {
	strList := strings.Split(strings.TrimSpace(line), fieldDelim)
	if len(strList) != numberOfFieldPasswdField {
		return nil, fmt.Errorf("Malformed content %s", line)
	}
	res := &User{
		Name:    strings.TrimSpace(strList[userNameOffset]),
		UID:     strings.TrimSpace(strList[uidOffset]),
		GID:     strings.TrimSpace(strList[userGidOffset]),
		Comment: strings.TrimSpace(strList[commentOffset]),
		Home:    strings.TrimSpace(strList[homeOffset]),
		Shell:   strings.TrimSpace(strList[shellOffset]),
	}
	return res, nil
}

func parsePasswdFile(passwdFilePath string) (*userData, error) {
	buf, err := ioutil.ReadFile(passwdFilePath)
	if err != nil {
		return nil, err
	}

	userDataObj := &userData{
		userMapByID:   make(map[string]*User),
		userMapByName: make(map[string][]*User),
		userSlice:     make([]*User, 0),
	}

	lines := strings.Split(strings.TrimSpace(string(buf)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// skip commented or empty lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		// parse the current line
		user, err := parseUser(line)
		if err != nil {
			return nil, err
		}
		userDataObj.userSlice = append(userDataObj.userSlice, user)
		userDataObj.userMapByID[user.UID] = user
		userDataObj.userMapByName[user.Name] = append(userDataObj.userMapByName[user.Name], user)
	}
	return userDataObj, nil
}
