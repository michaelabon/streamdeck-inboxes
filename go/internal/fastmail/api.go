package fastmail

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"golang.org/x/exp/slices"
)

type Settings struct {
	ApiToken string
}

func FetchUnseenCount(settings Settings) (uint, error) {
	if settings.ApiToken == "" {
		return 0, errors.New("missing ApiToken")
	}

	return getUnseenCount(settings)
}

type MailboxGetResponse struct {
	AccountId string
	State     string
	List      []Mailbox
}

type Mailbox struct {
	Id           string
	Name         string
	Role         string
	SortOrder    uint
	TotalEmails  uint
	UnreadEmails uint
}

type SessionResponse struct {
	PrimaryAccounts map[string]string
}

func makeRequest(url, method, bearer string, body io.Reader) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error while newing request: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+bearer)
	if method == "POST" {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error while doing request: %w", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("[fastmail]", "error while closing body:", err)
		}
	}(res.Body)

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error while reading body: %w", err)
	}

	return resBody, nil
}

func makeGetRequest(url, bearer string) ([]byte, error) {
	return makeRequest(url, "GET", bearer, nil)
}

func makePostRequest(url string, bearer string, body io.Reader) ([]byte, error) {
	return makeRequest(url, "POST", bearer, body)
}

func getUnseenCount(settings Settings) (uint, error) {
	sessionUrl := "https://api.fastmail.com/jmap/session"
	rawSessionResponse, err := makeGetRequest(sessionUrl, settings.ApiToken)
	if err != nil {
		return 0, fmt.Errorf("error while getting session: %w", err)
	}

	sessionResponse := &SessionResponse{}
	err = json.Unmarshal(rawSessionResponse, sessionResponse)
	if err != nil {
		return 0, fmt.Errorf("error while unmarshalling session response: %w", err)
	}
	accountId, ok := sessionResponse.PrimaryAccounts["urn:ietf:params:jmap:mail"]
	if !ok {
		return 0, fmt.Errorf(
			"error while retrieving primary account %v",
			sessionResponse.PrimaryAccounts,
		)
	}

	log.Println("[fastmail]", "successfully got accountId", accountId)

	apiUrl := "https://api.fastmail.com/jmap/api"
	apiBody := []byte(fmt.Sprintf(`{
		"using": ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
		"methodCalls": [[
			"Mailbox/get",
			{
				"accountId": "%s",
				"ids":       null
			},
			"0"
		]]
	}`, accountId))
	rawApiResponse, err := makePostRequest(apiUrl, settings.ApiToken, bytes.NewBuffer(apiBody))
	if err != nil {
		return 0, fmt.Errorf("error while posting Mailbox/get request: %w", err)
	}

	apiResponse := &apiResponse{}
	err = json.Unmarshal(rawApiResponse, apiResponse)
	if err != nil {
		return 0, fmt.Errorf("error while unmarshalling api response: %w", err)
	}

	invocation := apiResponse.MethodResponses[0]
	mailboxes := invocation.Args.List

	mailboxIdx := slices.IndexFunc(mailboxes, func(m Mailbox) bool {
		return m.Role == "inbox"
	})
	if mailboxIdx == -1 {
		return 0, fmt.Errorf("unable to find inbox in methodResponse %v", invocation)
	}

	return mailboxes[mailboxIdx].UnreadEmails, nil
}

const RefreshInterval = time.Minute

type apiResponse struct {
	MethodResponses []rawInvocation `json:"methodResponses"`
}

type rawInvocation struct {
	Name   string
	Args   MailboxGetResponse
	CallID string
}

func (i *rawInvocation) UnmarshalJSON(data []byte) error {
	var methodName, callId string
	var args MailboxGetResponse

	// Slice so we can detect invalid size.
	const correctSize = 3
	triplet := make([]json.RawMessage, 0, correctSize)
	if err := json.Unmarshal(data, &triplet); err != nil {
		return fmt.Errorf("error while unmarshalling triplet: %w", err)
	}
	if len(triplet) != correctSize {
		return fmt.Errorf(
			"jmap: malformed Invocation object, need exactly 3 elements, got %d, %v",
			len(triplet),
			triplet,
		)
	}

	if err := json.Unmarshal(triplet[0], &methodName); err != nil {
		return err
	}
	if err := json.Unmarshal(triplet[2], &callId); err != nil {
		return err
	}
	if err := json.Unmarshal(triplet[1], &args); err != nil {
		return err
	}

	i.Name = methodName
	i.CallID = callId
	i.Args = args

	return nil
}

func (i *rawInvocation) MarshalJSON() ([]byte, error) {
	return json.Marshal([3]interface{}{i.Name, i.Args, i.CallID})
}
