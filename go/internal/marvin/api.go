package marvin

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/exp/slices"
)

type Settings struct {
	Server   string
	Database string
	User     string
	Password string
}

const RefreshInterval = time.Minute

func FetchUnseenCount(settings *Settings) (uint, error) {
	if settings.Server == "" {
		return 0, errors.New("missing Server")
	}
	if settings.Database == "" {
		return 0, errors.New("missing Database")
	}
	if settings.User == "" {
		return 0, errors.New("missing User")
	}
	if settings.Password == "" {
		return 0, errors.New("missing Password")
	}

	return getUnseenCount(settings)
}

type response struct {
	Rows []struct {
		Doc task
	}
}

type task struct {
	Title     string
	DB        string `json:"db"`
	ParentID  string `json:"parentId"`
	Done      bool
	Recurring bool
}

func getUnseenCount(settings *Settings) (uint, error) {
	client := &http.Client{}

	marvinUrl, err := url.Parse(settings.Server)
	if err != nil {
		log.Println("[marvin] error while parsing url")
		return 0, err
	}
	marvinUrl = marvinUrl.JoinPath(settings.Database, "_all_docs")
	marvinUrl.Query().Add("include_docs", "true")

	req, err := http.NewRequest("GET", marvinUrl.String(), nil)
	if err != nil {
		log.Println("[marvin]", "error while newing request", err)
		return 0, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", makeBasicAuthorization(settings))

	res, err := client.Do(req)
	if err != nil {
		log.Println("[marvin]", "error while doing request", err)
		return 0, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("[marvin]", "error while closing body", err)
		}
	}(res.Body)

	resBody, err := io.ReadAll(res.Body)

	marvinResponse := &response{}
	err = json.Unmarshal(resBody, marvinResponse)
	if err != nil {
		log.Println("[marvin]", "error while unmarshalling session response", err)
		return 0, err
	}

	tasks := make([]task, len(marvinResponse.Rows))

	for i, r := range marvinResponse.Rows {
		tasks[i] = r.Doc
	}

	tasks = slices.DeleteFunc(tasks, func(task task) bool {
		return task.Title == "" ||
			task.DB != "Tasks" ||
			task.ParentID != "unassigned" ||
			task.Done ||
			task.Recurring
	})

	return uint(len(tasks)), nil
}

func makeBasicAuthorization(settings *Settings) string {
	decoded := settings.User + ":" + settings.Password
	encoded := base64.StdEncoding.EncodeToString([]byte(decoded))
	return fmt.Sprintf("Basic %s", encoded)
}
