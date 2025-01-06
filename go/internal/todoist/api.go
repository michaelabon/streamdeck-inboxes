package todoist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Settings struct {
	ApiToken string
}

const RefreshInterval = time.Minute

func FetchUnseenCount(settings *Settings) (uint, error) {
	if settings.ApiToken == "" {
		return 0, errors.New("missing ApiToken")
	}

	return getUnseenCount(settings)
}

type project struct {
	ID             string `json:"id"`
	IsInboxProject bool   `json:"is_inbox_project"`
	IsTeamInbox    bool   `json:"is_team_inbox"`
}

type task struct {
	IsCompleted bool `json:"is_completed"`
}

func getUnseenCount(settings *Settings) (uint, error) {
	client := &http.Client{}

	projectsUrl := "https://api.todoist.com/rest/v2/projects"

	projectsRequest, err := http.NewRequest("GET", projectsUrl, nil)
	if err != nil {
		return 0, fmt.Errorf("error while newing projects request: %w", err)
	}
	projectsRequest.Header.Add("Accept", "application/json")
	projectsRequest.Header.Add("Authorization", "Bearer "+settings.ApiToken)

	projectsResponse, err := client.Do(projectsRequest)
	if err != nil {
		return 0, fmt.Errorf("error while doing projects request: %w", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("[todoist]", "error while closing body", err)
		}
	}(projectsResponse.Body)

	projectsResponseBody, err := io.ReadAll(projectsResponse.Body)
	if err != nil {
		return 0, fmt.Errorf("error while reading projects response body: %w", err)
	}

	var projects []project
	err = json.Unmarshal(projectsResponseBody, &projects)
	if err != nil {
		return 0, fmt.Errorf(
			"error while unmarshalling projects response: %w",
			err,
		)
	}

	var inboxProjectIDs []string
	for _, p := range projects {
		if p.IsInboxProject || p.IsTeamInbox {
			inboxProjectIDs = append(inboxProjectIDs, p.ID)
		}
	}

	totalTasks := uint(0)
	for _, inboxProjectID := range inboxProjectIDs {
		tasksUrl := "https://api.todoist.com/rest/v2/tasks?project_id=" + inboxProjectID

		tasksRequest, err := http.NewRequest("GET", tasksUrl, nil)
		if err != nil {
			return 0, fmt.Errorf("error while calling NewRequest to GET tasks: %w", err)
		}
		tasksRequest.Header.Add("Accept", "application/json")
		tasksRequest.Header.Add("Authorization", "Bearer "+settings.ApiToken)

		tasksResponse, err := client.Do(tasksRequest)
		if err != nil {
			return 0, fmt.Errorf("error while doing GET tasks request: %w", err)
		}

		tasksResponseBody, err := io.ReadAll(tasksResponse.Body)
		if err != nil {
			closeErr := tasksResponse.Body.Close()
			if closeErr != nil {
				err = fmt.Errorf("error while closing task response body: %w", err)
			}

			return 0, fmt.Errorf("error while reading tasks body: %w", err)
		}

		var tasks []task
		err = json.Unmarshal(tasksResponseBody, &tasks)
		if err != nil {
			closeErr := tasksResponse.Body.Close()
			if closeErr != nil {
				err = fmt.Errorf("error while closing task response body: %w", err)
			}

			return 0, fmt.Errorf(
				"error while unmarshalling tasks response: %w",
				err,
			)
		}

		totalTasks += uint(len(tasks))

		closeErr := tasksResponse.Body.Close()
		if closeErr != nil {
			return 0, fmt.Errorf("error while closing task response body: %w", err)
		}
	}

	return totalTasks, err
}
