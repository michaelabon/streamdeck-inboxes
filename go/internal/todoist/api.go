package todoist

import (
	"encoding/json"
	"errors"
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
		log.Println("[todoist]", "error while newing projects request", err)

		return 0, err
	}
	projectsRequest.Header.Add("Accept", "application/json")
	projectsRequest.Header.Add("Authorization", "Bearer "+settings.ApiToken)

	projectsResponse, err := client.Do(projectsRequest)
	if err != nil {
		log.Println("[todoist]", "error while doing projects request", err)

		return 0, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("[todoist]", "error while closing body", err)
		}
	}(projectsResponse.Body)

	projectsResponseBody, err := io.ReadAll(projectsResponse.Body)
	if err != nil {
		log.Println(
			"[todoist]",
			"error while reading projects response body",
			err,
		)

		return 0, err
	}

	var projects []project
	err = json.Unmarshal(projectsResponseBody, &projects)
	if err != nil {
		log.Println(
			"[todoist]",
			"error while unmarshalling projects response",
			err,
			"\n",
			string(projectsResponseBody),
		)

		return 0, err
	}

	var inboxProjectIDs []string
	for _, p := range projects {
		if p.IsInboxProject || p.IsTeamInbox {
			inboxProjectIDs = append(inboxProjectIDs, p.ID)
		}
	}

	totalTasks := 0
	for _, inboxProjectID := range inboxProjectIDs {
		tasksUrl := "https://api.todoist.com/rest/v2/tasks?project_id=" + inboxProjectID

		tasksRequest, err := http.NewRequest("GET", tasksUrl, nil)
		if err != nil {
			log.Println("[todoist]", "error while newing tasks request", err)

			return 0, err
		}
		tasksRequest.Header.Add("Accept", "application/json")
		tasksRequest.Header.Add("Authorization", "Bearer "+settings.ApiToken)

		tasksResponse, err := client.Do(tasksRequest)
		if err != nil {
			log.Println("[todoist]", "error while doing tasks request", err)

			return 0, err
		}

		tasksResponseBody, err := io.ReadAll(tasksResponse.Body)
		if err != nil {
			log.Println("[todoist]", "error while reading tasks body")
			_ = tasksResponse.Body.Close()

			return 0, err
		}

		var tasks []task
		err = json.Unmarshal(tasksResponseBody, &tasks)
		if err != nil {
			log.Println(
				"[todoist]",
				"error while unmarshalling tasks response",
				err,
				"\n",
				string(tasksResponseBody),
			)
			_ = tasksResponse.Body.Close()

			return 0, err
		}

		totalTasks += len(tasks)

		_ = tasksResponse.Body.Close()
	}

	return uint(totalTasks), err
}
