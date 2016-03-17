package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/drone/routes"
	"github.com/imdario/mergo"
)

var proMap = make(map[string]profileStruct)

type profileStruct struct {
	Email      	   	string `json:"email"`
	Zip 			string `json:"zip"`
	Country       	string `json:"country"`
	Profession 		string `json:"profession"`
	FavoriteColor 	string `json:"favorite_color"`
	IsSmoking 		string `json:"is_smoking"`
	FavoriteSport 	string `json:"favorite_sport"`
	
	Food struct {
		Type         string `json:"type"`
		DrinkAlcohol string `json:"drink_alcohol"`
		} `json:"food"`

	Music struct {
		SpotifyUserID string `json:"spotify_user_id"`
		} `json:"music"`
	
	Movie struct {
		TvShows []string `json:"tv_shows"`
		Movies  []string `json:"movies"`
		} `json:"movie"`
	
	Travel     struct {
		Flight 	struct {
			Seat string `json:"seat"`
		} `json:"flight"`
	} `json:"travel"`
	
}

func main() {
	mux := routes.New()

	mux.Post("/profile", PostProfile)
	mux.Get("/profile/:email", GetProfile)
	mux.Put("/profile/:email", PutProfile)
	mux.Del("/profile/:email", DeleteProfile)

	http.Handle("/", mux)
	log.Println("Listening...")
	http.ListenAndServe(":3000", nil)
}

func PostProfile(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var p profileStruct
	err := decoder.Decode(&p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		panic("Some error occurred in decoding the JSON")
	}
	proMap[p.Email] = p
	w.WriteHeader(http.StatusCreated)
	w.WriteHeader(201)
	w.Write([]byte("Created"))
	w.Header().Set("Content-Type", "application/json")
}

func GetProfile(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	email := params.Get(":email")

	if val, ok := proMap[email]; ok {
		outputJSON, err := json.Marshal(val)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			panic("Some error occurred in parsing the JSON")
		}
		w.WriteHeader(200)
		w.Write(outputJSON)
		w.WriteHeader(http.StatusOK)
	}else{
		w.WriteHeader(404)
		w.WriteHeader(http.StatusInternalServerError)
	}
	
	w.Header().Set("Content-Type", "application/json")
}

func PutProfile(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	email := params.Get(":email")

	if val, ok := proMap[email]; ok {
		decoder := json.NewDecoder(r.Body)

		var p profileStruct
		err := decoder.Decode(&p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			panic("Some error occurred")
		}

		mergo.MergeWithOverwrite(&val, p)
		proMap[email] = val
		w.WriteHeader(http.StatusNoContent)
		w.WriteHeader(204)
		w.Write([]byte("No Content"))
		w.Header().Set("Content-Type", "application/json")

	} else {
		fmt.Println("Please Enter Valid Email ID")
	}
}

func DeleteProfile(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	email := params.Get(":email")
	delete(proMap, email)
	w.WriteHeader(204)
	w.Write([]byte("No Content"))
	w.Header().Set("Content-Type", "application/json")
}
