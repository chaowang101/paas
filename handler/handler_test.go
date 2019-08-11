package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chaowang101/paas/data"
)

type emptyPasswdMgr int

func (emptyPasswdMgr) Start() error {
	panic("implement me")
}

func (emptyPasswdMgr) Stop() {
	panic("implement me")
}

func (emptyPasswdMgr) GetAllUsers() []*data.User {
	return nil
}

func (emptyPasswdMgr) GetUserByQuery(name, uid, gid, comment, home, shell string) []*data.User {
	return nil
}

func (emptyPasswdMgr) GetUserByUID(uid string) *data.User {
	return nil
}

func (emptyPasswdMgr) GetAllGroups() []*data.Group {
	return nil
}

func (emptyPasswdMgr) GetGroupsByUID(uid string) []*data.Group {
	return nil
}

func (emptyPasswdMgr) GetGroupByQuery(name, gid string, members []string) []*data.Group {
	return nil
}

func (emptyPasswdMgr) GetGroupByGID(gid string) *data.Group {
	return nil
}

type dummyPasswdMgr int

func (dummyPasswdMgr) Start() error {
	panic("implement me")
}

func (dummyPasswdMgr) Stop() {
	panic("implement me")
}

var dummyUser []*data.User = []*data.User{
	&data.User{
		Name:    "root",
		UID:     "-1",
		GID:     "0",
		Comment: "System Administrator",
		Home:    "/var/root",
		Shell:   "/bin/sh",
	},
}

var dummyGroup []*data.Group = []*data.Group{
	&data.Group{
		Name:    "wheel",
		GID:     "0",
		Members: []string{"root", "root2"},
	},
}

func (dummyPasswdMgr) GetAllUsers() []*data.User {
	return dummyUser
}

func (dummyPasswdMgr) GetUserByQuery(name, uid, gid, comment, home, shell string) []*data.User {
	return dummyUser
}

func (dummyPasswdMgr) GetUserByUID(uid string) *data.User {
	return dummyUser[0]
}

func (dummyPasswdMgr) GetAllGroups() []*data.Group {
	return dummyGroup
}

func (dummyPasswdMgr) GetGroupsByUID(uid string) []*data.Group {
	return dummyGroup
}

func (dummyPasswdMgr) GetGroupByQuery(name, gid string, members []string) []*data.Group {
	return dummyGroup
}

func (dummyPasswdMgr) GetGroupByGID(gid string) *data.Group {
	return dummyGroup[0]
}

func assert(t *testing.T, condition bool) {
	if !condition {
		t.Fatal()
	}
}
func verifyResponseCode(handler http.Handler, path string, expectedStatus int, t *testing.T) *bytes.Buffer {
	rr := httptest.NewRecorder()
	req, err := http.NewRequest("GET", path, nil)
	assert(t, err == nil)
	handler.ServeHTTP(rr, req)
	assert(t, rr.Code == expectedStatus)
	return rr.Body
}

func verifyResponse(handler http.Handler, path string, expectedBuf *bytes.Buffer, expectedStatus int, t *testing.T) {
	buf := verifyResponseCode(handler, path, expectedStatus, t)
	if !bytes.Equal(buf.Bytes(), expectedBuf.Bytes()) {
		fmt.Println(buf.String())
		fmt.Println(expectedBuf.String())
	}
	assert(t, bytes.Equal(buf.Bytes(), expectedBuf.Bytes()))
}

func TestHandlerUserFunc(t *testing.T) {
	var dummyUserArrayJSON bytes.Buffer
	encoder := json.NewEncoder(&dummyUserArrayJSON)
	err := encoder.Encode(dummyUser)
	assert(t, err == nil)

	handler := New("", new(dummyPasswdMgr))

	var rr *httptest.ResponseRecorder

	// verify path "/users", "/users/query"
	for _, path := range []string{"/users", "/users/query?name=root"} {
		verifyResponse(handler, path, &dummyUserArrayJSON, http.StatusOK, t)
	}

	// verify path "user/{uid}"
	var dummyUserJSON bytes.Buffer
	encoder = json.NewEncoder(&dummyUserJSON)
	err = encoder.Encode(dummyUser[0])
	assert(t, err == nil)
	verifyResponse(handler, "/users/0", &dummyUserJSON, http.StatusOK, t)

	emptyHandler := New("", new(emptyPasswdMgr))
	rr = httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/users/-1", nil)
	assert(t, err == nil)
	emptyHandler.ServeHTTP(rr, req)
	assert(t, rr.Code == http.StatusNotFound)
}

func TestHandlerGroupFunc(t *testing.T) {
	handler := New("", new(dummyPasswdMgr))
	var dummyGroupArrayJSON bytes.Buffer
	encoder := json.NewEncoder(&dummyGroupArrayJSON)
	err := encoder.Encode(dummyGroup)
	assert(t, err == nil)

	for _, path := range []string{"/groups", "/groups/query?name=root", "/users/-1/groups"} {

		verifyResponse(handler, path, &dummyGroupArrayJSON, http.StatusOK, t)
	}

	var dummyGroupJSON bytes.Buffer
	encoder = json.NewEncoder(&dummyGroupJSON)
	err = encoder.Encode(dummyGroup[0])
	assert(t, err == nil)
	verifyResponse(handler, "/groups/0", &dummyGroupJSON, http.StatusOK, t)
}

func TestHandlerEmptyGroupFunc(t *testing.T) {
	emptyHandler := New("", new(emptyPasswdMgr))
	_ = verifyResponseCode(emptyHandler, "/group/0", http.StatusNotFound, t)

	for _, path := range []string{"/groups", "/groups/query?name=root"} {
		_ = verifyResponseCode(emptyHandler, path, http.StatusNoContent, t)
	}
}

func TestHandlerEmptyUserFunc(t *testing.T) {
	emptyHandler := New("", new(emptyPasswdMgr))
	_ = verifyResponseCode(emptyHandler, "/users/0", http.StatusNotFound, t)

	for _, path := range []string{"/users", "/users/query?name=root", "/users/-1/groups"} {
		_ = verifyResponseCode(emptyHandler, path, http.StatusNoContent, t)
	}
}
