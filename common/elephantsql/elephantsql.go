package elephantsql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
)

type ElephantSql struct {
	//ApiKey is the ElephantSQL API key. You can obtain yours from https://customer.elephantsql.com/team/api
	ApiKey string
	Client *http.Client
}

// These are the five ElephantSQL instance types available. TypeTinyTurtle is the free tier. Available instance types
// are described in detail on https://www.elephantsql.com/plans.html.
const (
	TypeTinyTurtle       = "turtle"
	TypePrettyPanda      = "panda"
	TypeHappyHippo       = "hippo"
	TypeEnormousElephant = "elephant"
	TypePuffyPigeon      = "pigeon"
)

// NewInstance creates a new ElephantSQL instance with the given name and of the given type. Regions may be an array
// containing a list of allowed regions (from https://customer.elephantsql.com/team/api). The actual region will be
// selected randomly. If regions is nil or empty, the region will be selected randomly from US regions.
func (e *ElephantSql) NewInstance(name, instance_type string, regions []string) (*Instance, error) {
	e.checkClient()

	reg := &regions
	if regions == nil || len(regions) == 0 {
		reg = &[]string{
			"amazon-web-services::us-east-1",
			"amazon-web-services::us-west-1",
			"amazon-web-services::us-west-2",
		}
	}

	body := bytes.NewBufferString(url.Values{
		"name":   []string{name},
		"plan":   []string{instance_type},
		"region": []string{(*reg)[rand.Intn(len(*reg))]},
	}.Encode())

	req, err := http.NewRequest(
		http.MethodPost,
		"https://customer.elephantsql.com/api/instances",
		body,
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.SetBasicAuth(e.ApiKey, "")

	res, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("server returned non-2XX status %d %q", res.StatusCode, res.Status)
	}

	jsonBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}

	i := &Instance{}
	if err := json.Unmarshal(jsonBytes, i); err != nil {
		return nil, fmt.Errorf("parsing rsponse body: %s", err)
	}

	if err := i.ParseUrl(); err != nil {
		return nil, fmt.Errorf("parsing instance url: %s", err)
	}

	return i, nil
}

// ListInstances returns a list of unhydrated/undetailed instances obtained from ElephantSQL; or an error if something
// went wrong during the request.
func (e *ElephantSql) ListInstances() ([]Instance, error) {
	e.checkClient()

	req, err := http.NewRequest(http.MethodGet, "https://customer.elephantsql.com/api/instances", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.SetBasicAuth(e.ApiKey, "")
	res, err := e.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dispatching request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("server returned non-2XX status %d %q", res.StatusCode, res.Status)
	}

	jsonBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}
	var instances = []Instance{}
	if err := json.Unmarshal(jsonBytes, instances); err != nil {
		return nil, fmt.Errorf("parsing response body: %s", err)
	}
	return instances, nil
}

// Enrich returns a new instance enriched with the instance details available from the API; most importantly the URL.
func (e *ElephantSql) Enrich(i *Instance) (*Instance, error) {
	e.checkClient()
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("https://customer.elephantsql.com/api/instances/%d", i.Id),
		nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %s", err)
	}
	req.SetBasicAuth(e.ApiKey, "")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dispatching request: %s", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 {
		return nil, fmt.Errorf("server returned non-2XX status %d %q", res.StatusCode, res.Status)
	}

	jsonBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %s", err)
	}
	var instances = []Instance{}
	if err := json.Unmarshal(jsonBytes, instances); err != nil {
		return nil, fmt.Errorf("parsing response body: %s", err)
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("instance with id %d not found", i.Id)
	}

	if err := instances[0].ParseUrl(); err != nil {
		return nil, err
	}

	return &instances[0], nil
}

// FindInstance obtains your list of ES instances and returns a hydrated/detailed Instance object for the named
// instances, if one exists. If no instance is found, the returned instance will be nil. Error is nil unless an error
// (other thanthe instance in question not existing) is returned.
func (e *ElephantSql) FindInstance(name string) (*Instance, error) {
	instances, err := e.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("listing instances: %s", err)
	}

	for _, i := range instances {
		if i.Name == name {
			return &i, nil
		}
	}
	return nil, nil
}

func (e *ElephantSql) checkClient() {
	if e.Client == nil {
		e.Client = http.DefaultClient
	}
}
