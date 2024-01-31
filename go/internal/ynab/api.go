package ynab

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/exp/slices"
)

type Settings struct {
	BudgetUuid          string `json:"budgetUuid"`
	PersonalAccessToken string `json:"apiToken"`
	NextAccountId       string `json:"-"`
}

const RefreshInternal = 2 * time.Minute

func FetchUnseenCountAndNextAccountId(settings *Settings) (uint, error) {
	if settings.BudgetUuid == "" {
		return 0, errors.New("missing BudgetUuid")
	}
	if settings.PersonalAccessToken == "" {
		return 0, errors.New("missing PersonalAccessToken")
	}

	return getUnseenCount(settings)
}

func getUnseenCount(settings *Settings) (uint, error) {
	transactionsUrl := fmt.Sprintf(
		"https://api.ynab.com/v1/budgets/%s/transactions?type=unapproved",
		settings.BudgetUuid,
	)

	rawTransactions, err := makeRequest(transactionsUrl, settings.PersonalAccessToken)
	if err != nil {
		log.Println("[ynab]", "error while getting transactions", err)
		return 0, err
	}

	type Transaction struct {
		AccountName string `json:"account_name"`
		AccountId   string `json:"account_id"`
	}

	type TransactionsResponse struct {
		Data struct {
			Transactions []Transaction
		}
	}

	transactions := &TransactionsResponse{}
	err = json.Unmarshal(rawTransactions, transactions)
	if err != nil {
		log.Println("[ynab]", "error while unmarshalling session response", err)
		return 0, err
	}

	result := slices.DeleteFunc(transactions.Data.Transactions, func(t Transaction) bool {
		return strings.HasPrefix(t.AccountName, "[D]") || strings.HasPrefix(t.AccountName, "[MD]")
	})

	if len(result) == 0 {
		return 0, nil
	}
	settings.NextAccountId = result[0].AccountId

	return uint(len(result)), nil
}

func makeRequest(url, bearer string) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("[ynab]", "error while newing request", err)
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+bearer)

	res, err := client.Do(req)
	if err != nil {
		log.Println("[ynab]", "error while doing request", err)
		return nil, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			log.Println("[ynab]", "error while closing body", err)
		}
	}(res.Body)

	resBody, err := io.ReadAll(res.Body)

	return resBody, nil
}
