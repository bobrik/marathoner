package marathoner

import (
	"encoding/json"
	"testing"
)

func responseToState(r string) (State, error) {
	mr := &marathonResponse{}

	err := json.Unmarshal([]byte(r), mr)
	if err != nil {
		return nil, err
	}

	return marathonResponseToState(mr)
}

const healthyAppWithoutHealthChecks = `
{
	"apps": [{
		"id": "/whatever",
		"ports": [1234],
		"tasks": [{
			"id": "whatever.361e84d1-b041-11e4-bc81-56847afe9799",
			"host": "web33",
			"ports": [
				31005
			],
			"startedAt": "2015-02-09T09:52:12.080Z",
			"stagedAt": "2015-02-09T09:51:25.529Z",
			"version": "2015-02-09T09:51:20.692Z",
			"appId": "/whatever"
		}]
	}]
}
`

func TestAppWithoutHealthChecks(t *testing.T) {
	s, err := responseToState(healthyAppWithoutHealthChecks)
	if err != nil {
		t.Fatal(err)
	}

	if len(s["/whatever"].Tasks) != 1 {
		t.Fatalf("found %d tasks when expected %d", len(s["/whatever"].Tasks), 1)
	}
}

const healthyAppWithGoodHealthChecks = `
{
	"apps": [{
		"id": "/whatever",
		"ports": [1234],
		"tasks": [{
			"id": "whatever.361e84d1-b041-11e4-bc81-56847afe9799",
			"host": "web33",
			"ports": [
				31005
			],
			"startedAt": "2015-02-09T09:52:12.080Z",
			"stagedAt": "2015-02-09T09:51:25.529Z",
			"version": "2015-02-09T09:51:20.692Z",
			"appId": "/whatever",
			"healthCheckResults": [{
				"alive": true,
				"consecutiveFailures": 0,
				"firstSuccess": "2015-02-09T09:54:41.307Z",
				"lastFailure": null,
				"lastSuccess": "2015-02-09T09:54:41.307Z",
				"taskId": "whatever.361e84d1-b041-11e4-bc81-56847afe9799"
			}]
		}]
	}]
}
`

func TestAppWithGoodHealthChecks(t *testing.T) {
	s, err := responseToState(healthyAppWithGoodHealthChecks)
	if err != nil {
		t.Fatal(err)
	}

	if len(s["/whatever"].Tasks) != 1 {
		t.Fatalf("found %d tasks when expected %d", len(s["/whatever"].Tasks), 1)
	}
}

const healthyAppWithBadHealthChecks = `
{
	"apps": [{
		"id": "/whatever",
		"ports": [1234],
		"tasks": [{
			"id": "whatever.361e84d1-b041-11e4-bc81-56847afe9799",
			"host": "web33",
			"ports": [
				31005
			],
			"startedAt": "2015-02-09T09:52:12.080Z",
			"stagedAt": "2015-02-09T09:51:25.529Z",
			"version": "2015-02-09T09:51:20.692Z",
			"appId": "/whatever",
			"healthCheckResults": [{
				"alive": false,
				"consecutiveFailures": 0,
				"firstSuccess": null,
				"lastFailure": null,
				"lastSuccess": null,
				"taskId": "whatever.361e84d1-b041-11e4-bc81-56847afe9799"
			}]
		}]
	}]
}
`

func TestAppWithBadHealthChecks(t *testing.T) {
	s, err := responseToState(healthyAppWithBadHealthChecks)
	if err != nil {
		t.Fatal(err)
	}

	if len(s["/whatever"].Tasks) != 0 {
		t.Fatalf("found %d tasks when expected %d", len(s["/whatever"].Tasks), 0)
	}
}

// see https://github.com/mesosphere/marathon/issues/1106
const healthyAppWithNullHealthChecks = `
{
	"apps": [{
		"id": "/whatever",
		"ports": [1234],
		"tasks": [{
			"id": "whatever.361e84d1-b041-11e4-bc81-56847afe9799",
			"host": "web33",
			"ports": [
				31005
			],
			"startedAt": "2015-02-09T09:52:12.080Z",
			"stagedAt": "2015-02-09T09:51:25.529Z",
			"version": "2015-02-09T09:51:20.692Z",
			"appId": "/whatever",
			"healthCheckResults": [
				null
			]
		}]
	}]
}
`

func TestAppWithNullHealthChecks(t *testing.T) {
	s, err := responseToState(healthyAppWithNullHealthChecks)
	if err != nil {
		t.Fatal(err)
	}

	if len(s["/whatever"].Tasks) != 1 {
		t.Fatalf("found %d tasks when expected %d", len(s["/whatever"].Tasks), 1)
	}
}
