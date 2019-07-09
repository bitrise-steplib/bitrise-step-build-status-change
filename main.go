package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/log"
	"github.com/google/go-querystring/query"
	"github.com/pkg/errors"
)

const (
	baseURL                   = "https://api.bitrise.io/v0.1"
	buildTypeTag              = "tag"
	buildTypePush             = "push"
	buildTypeManual           = "manual"
	buildTypePullRequest      = "pull-request"
	envKeyPreviousBuildStatus = "PREVIOUS_BUILD_STATUS"
	envKeyBuildStatusChanged  = "BUILD_STATUS_CHANGED"
	statusTextSuccessfulBuild = "success"
)

type config struct {
	AppSlug             string          `env:"BITRISE_APP_SLUG,required"`
	BuildSlug           string          `env:"BITRISE_BUILD_SLUG,required"`
	BuildStatus         string          `env:"BITRISE_BUILD_STATUS,required"`
	PreviousBuildStatus string          `env:"PREVIOUS_BUILD_STATUS"`
	AccessToken         stepconf.Secret `env:"access_token,required"`
}

func (cfg config) getBuild() (build, error) {
	url := baseURL + "/apps/" + cfg.AppSlug + "/builds/" + cfg.BuildSlug

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return build{}, err
	}
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", string(cfg.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return build{}, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return build{}, err
	}
	if resp.StatusCode != 200 {
		return build{}, fmt.Errorf("invalid response status code: %d\nbody: %s", resp.StatusCode, string(body))
	}

	var b struct{ Data build }
	if err := json.Unmarshal(body, &b); err != nil {
		return build{}, errors.Wrap(err, string(body))
	}
	return b.Data, nil
}

func (cfg config) getBuilds(f filter) (builds, error) {
	url := baseURL + "/apps/" + cfg.AppSlug + "/builds"

	f.SortBy = "created_at"
	queryParams, err := query.Values(f)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.URL.RawQuery = queryParams.Encode()
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", string(cfg.AccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return builds{}, fmt.Errorf("invalid response status code: %d\nbody: %s", resp.StatusCode, string(body))
	}

	var builds struct {
		Data []build `json:"data"`
	}
	if err := json.Unmarshal(body, &builds); err != nil {
		return nil, errors.Wrap(err, string(body))
	}
	return builds.Data, nil
}

type filter struct {
	Before           int    `url:"before,omitempty"`
	After            int    `url:"after,omitempty"`
	Limit            int    `url:"limit,omitempty"`
	Branch           string `url:"branch,omitempty"`
	SortBy           string `url:"sort_by,omitempty"`
	Workflow         string `url:"workflow,omitempty"`
	PullRequestID    int    `url:"pull_request_id,omitempty"`
	TriggerEventType string `url:"trigger_event_type,omitempty"`
}

func (f filter) String() string {
	return fmt.Sprintf("- branch: %s\n- workflow: %s\n- pull request ID: %d\n- event type: %s\n", f.Branch, f.Workflow, f.PullRequestID, f.TriggerEventType)
}

type build struct {
	Tag                     string    `json:"tag"`
	Slug                    string    `json:"slug"`
	Branch                  string    `json:"branch"`
	Status                  int       `json:"status"`
	CommitHash              string    `json:"commit_hash"`
	StatusText              string    `json:"status_text"`
	TriggeredAt             time.Time `json:"triggered_at"`
	BuildNumber             int64     `json:"build_number"`
	PullRequestID           *int64    `json:"pull_request_id"`
	TriggeredWorkflow       string    `json:"triggered_workflow"`
	PullRequestTargetBranch string    `json:"pull_request_target_branch"`
}

func (build build) generateFilter() filter {
	f := filter{
		Workflow: build.TriggeredWorkflow,
		Branch:   build.Branch,
		Before:   int(build.TriggeredAt.Unix()),
		After:    int(build.TriggeredAt.Unix()) - 24*60*60,
	}
	switch build.buildType() {
	case buildTypeTag:
		f.TriggerEventType = "tag"
	case buildTypePullRequest:
		f.TriggerEventType = "pull-request"
		f.PullRequestID = int(*build.PullRequestID)
	default: // website handles manual as push
		f.TriggerEventType = "push"
	}
	return f
}

func (build build) buildType() string {
	switch {
	case build.Tag != "":
		return buildTypeTag
		case (build.PullRequestID != nil && *build.PullRequestID > 0) || (build.PullRequestTargetBranch != ""):
		return buildTypePullRequest
	case build.CommitHash == "":
		return buildTypeManual
	default:
		return buildTypePush
	}
}

func (build build) equivalent(pair build) bool {
	return build.Branch == pair.Branch &&
		build.TriggeredWorkflow == pair.TriggeredWorkflow &&
		build.buildType() == pair.buildType()
}

type builds []build

func (builds builds) previous(actualBuild build) (build, error) {
	// builds is a sorted list because the aPI call has the filter set to do the sorting: SortBy = "created_at"
	for _, build := range builds {
		// skip if found the current build or any in-progress ones
		if build.BuildNumber == actualBuild.BuildNumber || build.Status == 0 {
			continue
		}
		if build.equivalent(actualBuild) {
			return build, nil
		}
	}
	return build{}, fmt.Errorf("no equivalent build found")
}

func failf(f string, args ...interface{}) {
	log.Errorf(f, args...)
	os.Exit(1)
}

func main() {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(cfg)
	fmt.Println()

	if cfg.PreviousBuildStatus != "" {
		log.Infof("Exporting environment variables:")
		currentBuildFailed, previousBuildFailed := cfg.BuildStatus != "0", cfg.PreviousBuildStatus != statusTextSuccessfulBuild
		changed := fmt.Sprintf("%t", currentBuildFailed != previousBuildFailed)
		if err := tools.ExportEnvironmentWithEnvman(envKeyBuildStatusChanged, changed); err != nil {
			failf("failed to export env: %s, error: %s", envKeyBuildStatusChanged, err)
		}
		log.Printf("- %s=%s", envKeyBuildStatusChanged, changed)
		log.Donef("- Done")
		return
	}

	log.Infof("Getting current build")
	currentBuild, err := cfg.getBuild()
	if err != nil {
		failf("- Failed to get current build, error: %s", err)
	}
	log.Printf("Build info:")
	filter := currentBuild.generateFilter()
	log.Printf("%s", filter)
	log.Donef("- Done")
	fmt.Println()

	log.Infof("Getting similar builds")
	similarBuilds, err := cfg.getBuilds(filter)
	if err != nil {
		failf("- Failed to get similar builds, error: %s", err)
	}
	log.Printf("%d builds found", len(similarBuilds))
	log.Donef("- Done")
	fmt.Println()

	log.Infof("Looking for previous build")
	matchingPreviousBuild, err := similarBuilds.previous(currentBuild)
	if err != nil {
		failf("- Failed to find previous build, error: %s", err)
	}
	log.Donef("- Found: (#%d) %s: %s", matchingPreviousBuild.BuildNumber, matchingPreviousBuild.Slug, matchingPreviousBuild.StatusText)
	fmt.Println()

	previousBuildFailed := matchingPreviousBuild.StatusText != statusTextSuccessfulBuild

	log.Infof("Exporting environment variables:")
	if err := tools.ExportEnvironmentWithEnvman(envKeyPreviousBuildStatus, matchingPreviousBuild.StatusText); err != nil {
		failf("failed to export env: %s, error: %s", envKeyPreviousBuildStatus, err)
	}
	log.Printf("- %s=%s", envKeyPreviousBuildStatus, matchingPreviousBuild.StatusText)

	currentBuildFailed := cfg.BuildStatus != "0"
	changed := fmt.Sprintf("%t", currentBuildFailed != previousBuildFailed)
	if err := tools.ExportEnvironmentWithEnvman(envKeyBuildStatusChanged, changed); err != nil {
		failf("failed to export env: %s, error: %s", envKeyBuildStatusChanged, err)
	}
	log.Printf("- %s=%s", envKeyBuildStatusChanged, changed)
	log.Donef("- Done")
}
