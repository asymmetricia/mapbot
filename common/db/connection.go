package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/pdbogen/mapbot/common/db/schema"
	"strings"
	"errors"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

var Instance *sql.DB

type instance struct {
	Id int `json:"id"`
	Plan string `json:"plan,omitempty"`
	Region string `json:"region,omitempty"`
	Name string `json:"name,omitempty"`
	Url string `json:"url,omitempty"`
}

func listInstances(key string) ([]instance,error) {
	req, err := http.NewRequest(http.MethodGet, "https://customer.elephantsql.com/api/instances", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.SetBasicAuth(key, "")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dispatching request: %s", err)
	}
	defer res.Body.Close()
	jsonBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}
	var instances = []instance{}
	if err := json.Unmarshal(jsonBytes, instances); err != nil {
		return nil, fmt.Errorf("parsing response body: %s", err)
	}
	return instances, nil
}

func (i *instance) Detailed(key string) (*instance, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://customer.elephantsql.com/api/instances/%d", i.Id),
		nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.SetBasicAuth(key, "")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dispatching request: %s", err)
	}
	defer res.Body.Close()
	jsonBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}
	var instances = []instance{}
	if err := json.Unmarshal(jsonBytes, instances); err != nil {
		return nil, fmt.Errorf("parsing response body: %s", err)
	}
	if len(instances) == 0 {
		return nil, fmt.Printf("instance with id %d not found", i.Id)
	}
	return instances[0], nil
}

func OpenElephant(key, instance_type string) (*sql.DB, error) {
	if key == "" {
		return nil, errors.New("key cannot be blank")
	}
	if instance_type == "" {
		return nil, errors.New("instance type cannot be blank")
	}

	instances, err := listInstances(key)
	if err != nil {
		return nil, fmt.Errorf("checking for existing instance: %s")
	}

	var instance *instance
	for _, i := range instances {
		if i.Name == "mapbot" {
			instance = &i
			break
		}
	}
	if instance != nil {
		detailed, err := instance.Detailed(key)
		if err != nil {
			return nil, fmt.Errorf("obtaining instance id %d URI: %s", instance.Id, err)
		}
		parts := strings.Split(detailed, "/")
		
		// connect to found instance
	} else {
		// create instance
	}
}

func Open(host, user, pass, db string, port int, reset bool) (*sql.DB, error) {
	dbConn, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"dbname=%s user=%s password=%s host=%s port=%d sslmode=verify-full",
			sanitize(db),
			sanitize(user),
			sanitize(pass),
			sanitize(host),
			port,
		),
	)
	if err != nil {
		return nil, err
	}
	if err := dbConn.Ping(); err != nil {
		return nil, err
	}

	if reset {
		if err := schema.Reset(dbConn); err != nil {
			return nil, err
		}
	}

	if err := schema.Apply(dbConn); err != nil {
		return nil, err
	}

	Instance = dbConn

	return dbConn, nil
}

func sanitize(in string) (out string) {
	return fmt.Sprintf("'%s'", strings.Replace(in, "'", "\\'", -1))
}
