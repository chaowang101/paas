package handler

import (
	"encoding/json"
	"fmt"
	"github.com/chaowang101/paas/data"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

const (
	//query fields
	qryName = "name"
	qryGID  = "gid"
	qryUID  = "uid"

	groupQryMember = "member"
	userQryComment = "comment"
	userQryHome    = "home"
	userQryShell   = "shell"

	userPath  = "/users"
	queryPath = "/query"
	groupPath = "/groups"

	// -? means one or zero occurrences of "-" to handle negative number
	userIDPath     = userPath + "/{uid:-?[0-9]+}"
	groupIDPath    = groupPath + "/{gid:-?[0-9]+}"
	groupByUIDPath = userPath + "/{uid:-?[0-9]+}" + groupPath
)

type handlerFunc func(dataMgr data.Manager, writer http.ResponseWriter, request *http.Request)

type handlerObj struct {
	handler handlerFunc
	query   bool
}

// every handler must register in this map
var getHandlerMap = map[string]*handlerObj{
	userPath:              &handlerObj{handler: usersAll},
	userPath + queryPath:  &handlerObj{handler: usersByQuery, query: true},
	userIDPath:            &handlerObj{handler: usersByUID},
	groupByUIDPath:        &handlerObj{handler: groupsByUID},
	groupPath:             &handlerObj{handler: groupsAll},
	groupPath + queryPath: &handlerObj{handler: groupsByQuery, query: true},
	groupIDPath:           &handlerObj{handler: groupsByGID},
}

// New returns a http.Handler that server the data from dataMgr
func New(domain string, dataMgr data.Manager) http.Handler {
	handler := mux.NewRouter()

	for path, obj := range getHandlerMap {
		curObj := obj
		route := handler.HandleFunc(path, func(writer http.ResponseWriter, request *http.Request) {
			// NOTE: more middleware should be called here
			// TODO: Those logs might be too verbose.
			log.Printf("Request %s from %v starts", request.RequestURI, request.RemoteAddr)
			curObj.handler(dataMgr, writer, request)
			log.Printf("Request %s from %v ends", request.RequestURI, request.RemoteAddr)
		}).Methods("GET").Queries()
		if obj.query {
			route = route.Queries()
		}
		if len(domain) > 0 {
			route.Host(domain)
		}
	}

	return handler
}

func encodeJSON(w http.ResponseWriter, v interface{}, errMsg string) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		log.Printf("%s with err: %s\n", errMsg, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func usersAll(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	users := dataMgr.GetAllUsers()
	if len(users) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSON(w, users, "Fail to encode the result of all users")
}

func usersByQuery(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	name := v.Get(qryName)
	uid := v.Get(qryUID)
	gid := v.Get(qryGID)
	comment := v.Get(userQryComment)
	home := v.Get(userQryHome)
	shell := v.Get(userQryShell)
	users := dataMgr.GetUserByQuery(name, uid, gid, comment, home, shell)
	if len(users) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSON(w, users, "Fail to encode the result of user query")
}

func usersByUID(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)[qryUID]
	user := dataMgr.GetUserByUID(uid)
	if user == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	encodeJSON(w, user, fmt.Sprintf("Fail to encode the result of user with UID %s", uid))
}

func groupsAll(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	groups := dataMgr.GetAllGroups()
	if len(groups) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSON(w, groups, "Fail to encode the result of all groups")
}

func groupsByUID(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	uid := mux.Vars(r)[qryUID]
	groups := dataMgr.GetGroupsByUID(uid)
	if len(groups) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	encodeJSON(w, groups, fmt.Sprintf("Fail to encode the result of group with UID %s", uid))
}

func groupsByQuery(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	v := r.URL.Query()
	name := v.Get(qryName)
	gid := v.Get(qryGID)
	members := v[groupQryMember]
	groups := dataMgr.GetGroupByQuery(name, gid, members)
	if len(groups) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	encodeJSON(w, groups, "Fail to encode the result of group query")
}

func groupsByGID(dataMgr data.Manager, w http.ResponseWriter, r *http.Request) {
	gid := mux.Vars(r)[qryGID]
	group := dataMgr.GetGroupByGID(gid)

	if group == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	encodeJSON(w, group, fmt.Sprintf("Fail to encode the result of group with GID %s", gid))
}
