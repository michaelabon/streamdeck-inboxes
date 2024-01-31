package fastmail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	"ca.michaelabon.inboxes/internal/display"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/samwho/streamdeck"
	"golang.org/x/exp/slices"
)

type Settings struct {
	ApiToken string
}

const (
	DefaultState = 0
	GoldState    = 1
)

func FetchAndUpdate(client *streamdeck.Client, ctx context.Context, settings *Settings) error {
	unseenCount, err := getUnseenCount(settings)
	if err != nil {
		log.Println("error while fetching unseen count", err)
		newErr := client.SetTitle(ctx, display.PadRight("!"), streamdeck.HardwareAndSoftware)
		if newErr != nil {
			log.Println("error while settings icon title with error", err)
			return newErr
		}
		return err
	}

	if unseenCount == 0 {
		err = client.SetState(ctx, GoldState)
		if err != nil {
			log.Println("error while setting state", err)
			return err
		}
		err = client.SetTitle(ctx, "", streamdeck.HardwareAndSoftware)
		if err != nil {
			log.Println("error while setting icon title with unseen count", err)
			return err
		}
	} else {
		err = client.SetState(ctx, DefaultState)
		if err != nil {
			log.Println("error while setting state", err)
			return err
		}
		err = client.SetTitle(ctx, display.PadRight(strconv.Itoa(int(unseenCount))), streamdeck.HardwareAndSoftware)
		if err != nil {
			log.Println("error while setting icon title with unseen count", err)
			return err
		}
	}

	return nil
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

func makeRequest(url, method, bearer string, body interface{}) ([]byte, error) {
	client := retryablehttp.NewClient()
	req, err := retryablehttp.NewRequest(method, url, body)
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

	log.Println("BODY RETURNED", string(resBody))
	return resBody, nil
}

func makeGetRequest(url, bearer string) ([]byte, error) {
	return makeRequest(url, "GET", bearer, nil)
}

func makePostRequest(url string, bearer string, body interface{}) ([]byte, error) {
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
		log.Println("[fastmail]", "error while retrieving primary account", sessionResponse.PrimaryAccounts)
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
	rawApiResponse, err := makePostRequest(apiUrl, settings.ApiToken, apiBody)
	if err != nil {
		log.Println("[fastmail]", "error while posting request", err)
		return 0, err
	}

	apiResponse := &ApiResponse{}
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

type ApiResponse struct {
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

	log.Println("[fastmail] [rawInvocation.UnmarshalJSON] data", string(data))

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
