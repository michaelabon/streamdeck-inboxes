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

func FetchUnseenCount(settings *Settings) (uint, error) {
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
		log.Println("[fastmail]", "error while newing request", err)
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+bearer)
	if method == "POST" {
		req.Header.Add("Content-Type", "application/json")
	}

	res, err := client.Do(req)
	if err != nil {
		log.Println("[fastmail]", "error while doing request", err)
		return nil, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("[fastmail]", "error while closing body", err)
		}
	}(res.Body)

	resBody, err := io.ReadAll(res.Body)

	return resBody, nil
}

func makeGetRequest(url, bearer string) ([]byte, error) {
	return makeRequest(url, "GET", bearer, nil)
}

func makePostRequest(url string, bearer string, body io.Reader) ([]byte, error) {
	return makeRequest(url, "POST", bearer, body)
}

func getUnseenCount(settings *Settings) (uint, error) {
	sessionUrl := "https://api.fastmail.com/jmap/session"
	rawSessionResponse, err := makeGetRequest(sessionUrl, settings.ApiToken)
	if err != nil {
		log.Println("[fastmail]", "error while getting session", err)
		return 0, err
	}

	sessionResponse := &SessionResponse{}
	err = json.Unmarshal(rawSessionResponse, sessionResponse)
	if err != nil {
		log.Println("[fastmail]", "error while unmarshalling session response", err)
		return 0, err
	}
	accountId, ok := sessionResponse.PrimaryAccounts["urn:ietf:params:jmap:mail"]
	if !ok {
		log.Println(
			"[fastmail]",
			"error while retrieving primary account",
			sessionResponse.PrimaryAccounts,
		)
		return 0, errors.New("error while retrieving primary account")
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
		log.Println("[fastmail]", "error while posting request", err)
		return 0, err
	}

	apiResponse := &apiResponse{}
	err = json.Unmarshal(rawApiResponse, apiResponse)
	if err != nil {
		log.Println("[fastmail]", "error while unmarshalling session response", err)
		return 0, err
	}

	invocation := apiResponse.MethodResponses[0]
	mailboxes := invocation.Args.List

	mailboxIdx := slices.IndexFunc(mailboxes, func(m Mailbox) bool {
		return m.Role == "inbox"
	})
	if mailboxIdx == -1 {
		return 0, fmt.Errorf("unable to find inbox in methodResponse")
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
	triplet := make([]json.RawMessage, 0, 3)
	if err := json.Unmarshal(data, &triplet); err != nil {
		return err
	}
	if len(triplet) != 3 {
		return errors.New("jmap: malformed Invocation object, need exactly 3 elements")
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

func (i rawInvocation) MarshalJSON() ([]byte, error) {
	return json.Marshal([3]interface{}{i.Name, i.Args, i.CallID})
}
