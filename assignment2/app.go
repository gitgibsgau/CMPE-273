package main

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"github.com/drone/routes"
	"github.com/mkilling/goejdb"
	"github.com/naoina/toml"
	"labix.org/v2/mgo/bson"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
)

type tomlConfig struct {
	Database struct {
		File_name string
		Port_num  int
	}
	Replication struct {
		Rpc_server_port_num int
		Replica             []string
	}
}

type Listener int

var oMap map[string]string

var config tomlConfig

func ConfigureTOML(tomlFile string) {
	f, err := os.Open(tomlFile)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	buf, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	if err := toml.Unmarshal(buf, &config); err != nil {
		panic(err)
	}
}

func LaunchRpcServer() {
	n, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+strconv.Itoa(config.Replication.Rpc_server_port_num))
	if err != nil {
		panic(err)
	}

	inbound, err := net.ListenTCP("tcp", n)
	if err != nil {
		panic(err)
	}

	log.Println("RPC Server Listening to... http://0.0.0.0:" + strconv.Itoa(config.Replication.Rpc_server_port_num))
	listener := new(Listener)
	rpc.Register(listener)
	rpc.Accept(inbound)

}

func main() {
	tomlFile := os.Args[1]
	ConfigureTOML(tomlFile)

	oMap = make(map[string]string)

	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
	gob.Register(bson.ObjectId(""))

	go LaunchRpcServer()

	mux := routes.New()

	mux.Post("/profile", PostProfile)
	mux.Get("/profile/:email", GetProfile)
	mux.Put("/profile/:email", PutProfile)
	mux.Del("/profile/:email", DeleteProfile)

	http.Handle("/", mux)
	log.Println("Listening at http://localhost:" + strconv.Itoa(config.Database.Port_num))
	http.ListenAndServe(":"+strconv.Itoa(config.Database.Port_num), nil)
}

func PostProfile(w http.ResponseWriter, r *http.Request) {
	val, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	var profile map[string]interface{}
	if err := json.Unmarshal(val, &profile); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) 
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	coll, _ := jb.CreateColl("profiles", nil)

	email := profile["email"].(string)
	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) != 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusConflict)
		jb.Close()
		return
	}
	bsProfile, _ := bson.Marshal(profile)
	oMap[email], _ = coll.SaveBson(bsProfile)

	jb.Close()

	UpdateReplica("POST", profile)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
}

func GetProfile(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	email := params.Get(":email")

	jb, err := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if err != nil {
		os.Exit(1)
	}

	coll, _ := jb.GetColl("profiles")

	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		jb.Close()
		return
	} else { 
		for _, bs := range res {
			var profile map[string]interface{}
			bson.Unmarshal(bs, &profile)
			delete(profile, "_id") 
			if err := json.NewEncoder(w).Encode(profile); err != nil {
				panic(err)
			}
		}
	}

	jb.Close()

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}

func PutProfile(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	email := params.Get(":email")

	val, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	if err := r.Body.Close(); err != nil {
		panic(err)
	}

	var newProfile map[string]interface{}
	if err := json.Unmarshal(val, &newProfile); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(422) 
		if err := json.NewEncoder(w).Encode(err); err != nil {
			panic(err)
		}
	}

	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	coll, _ := jb.GetColl("profiles")

	var oldProfile map[string]interface{} // Used later in else
	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		jb.Close()
		return
	} else { 
		for _, bs := range res {
			bson.Unmarshal(bs, &oldProfile)
		}
	}

	for key, _ := range newProfile {
		for k, _ := range oldProfile {
			if key == k {
				oldProfile[k] = newProfile[key]
			}
		}
	}
	fmt.Println(oldProfile)
	fmt.Println(newProfile)

	coll.RmBson(oMap[email])
	delete(oMap, email)

	bsProfile, _ := bson.Marshal(oldProfile)
	oMap[email], _ = coll.SaveBson(bsProfile)

	jb.Close()

	UpdateReplica("PUT", oldProfile)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNoContent)

}

func DeleteProfile(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	email := params.Get(":email")

	jb, err := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER)
	if err != nil {
		os.Exit(1)
	}

	coll, _ := jb.GetColl("profiles")

	res, _ := coll.Find(`{"email": "` + email + `"}`)
	if len(res) == 0 {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		jb.Close()
		return
	}

	coll.RmBson(oMap[email])
	delete(oMap, email)

	jb.Close()

	var profile = make(map[string]interface{})
	profile["email"] = email
	UpdateReplica("DEL", profile)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNoContent)

}


func (l *Listener) ReplicatePost(profile map[string]interface{}, ack *bool) error {
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	coll, _ := jb.CreateColl("profiles", nil)

	email := profile["email"].(string)
	bsProfile, _ := bson.Marshal(profile)
	oMap[email], _ = coll.SaveBson(bsProfile)

	jb.Close()

	return nil
}

func (l *Listener) ReplicatePut(profile map[string]interface{}, ack *bool) error {
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER|goejdb.JBOCREAT)
	if dbErr != nil {
		os.Exit(1)
	}

	coll, _ := jb.GetColl("profiles")

	email := profile["email"].(string)
	coll.RmBson(oMap[email])
	delete(oMap, email)

	bsProfile, _ := bson.Marshal(profile)
	oMap[email], _ = coll.SaveBson(bsProfile)

	jb.Close()

	return nil
}

func (l *Listener) ReplicateDel(profile map[string]interface{}, ack *bool) error {
	jb, dbErr := goejdb.Open(
		config.Database.File_name, goejdb.JBOWRITER)
	if dbErr != nil {
		os.Exit(1)
	}

	coll, _ := jb.GetColl("profiles")

	email := profile["email"].(string)
	coll.RmBson(oMap[email])
	delete(oMap, email)

	jb.Close()

	return nil
}

func UpdateReplica(method string, profile map[string]interface{}) {

	numReplica := len(config.Replication.Replica)
	for i := 0; i < numReplica; i++ {

		client, err := rpc.Dial(
			"tcp", config.Replication.Replica[i])
		if err != nil {
			panic(err)
		}

		if method == "POST" {
			var reply bool
			err = client.Call("Listener.ReplicatePost", profile, &reply)
			if err != nil {
				log.Fatal(err)
			}
		} else if method == "PUT" {
			var reply bool
			err = client.Call("Listener.ReplicatePut", profile, &reply)
			if err != nil {
				log.Fatal(err)
			}
		} else if method == "DEL" {
			var reply bool
			err = client.Call("Listener.ReplicateDel", profile, &reply)
			if err != nil {
				log.Fatal(err)
			}
		}
		log.Println("Calling Replica at... http://" + config.Replication.Replica[i])
	}
}

