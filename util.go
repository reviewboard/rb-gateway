package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

var repos map[string]Repository
var port int
var username string
var password string

// LoadConfig loads the configuration file at the specified path.
// The configuration file is a json file containing repositories with their
// corresponding file paths, the port number to run the server on, and the
// authentication information for the admin user. Repository names specified
// in the configuration file must be unique; they do not need to be the actual
// repository name.
func LoadConfig(path string) {
	type Repo struct {
		Name string
		Path string
		Scm  string
	}

	type Configuration struct {
		Port         int
		Username     string
		Password     string
		Repositories []Repo
	}

	var conf Configuration

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("ReadFile: ", err,
			"\nDid you set up ", path, " correctly?")
	}

	if json.Unmarshal(content, &conf) != nil {
		log.Fatal("Unmarshal: ", err,
			"\nDid you set up ", path, " correctly?")
	}

	port = conf.Port
	username = conf.Username
	password = conf.Password

	newRepos := make(map[string]Repository)

	for _, r := range conf.Repositories {
		switch r.Scm {
		case "git":
			newRepos[r.Name] = &GitRepository{RepositoryInfo{r.Name, r.Path}}
		default:
			log.Println("Repository SCM type is not defined: ", r.Scm)
		}
	}

	repos = newRepos
}

// GetRepository returns the repository based on the name specified in
// config.json.
func GetRepository(name string) Repository {
	return repos[name]
}

// GetPort returns the port where rb-gateway is running.
func GetPort() int {
	return port
}

// GetUsername returns the admin username.
func GetUsername() string {
	return username
}

// GetPassword returns the admin password.
func GetPassword() string {
	return password
}
