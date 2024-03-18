package gmail

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Settings struct {
	Username string
	Password string
}

func FetchUnseenCount(settings Settings) (uint, error) {
	if settings.Username == "" {
		return 0, errors.New("missing Username")
	}
	if settings.Password == "" {
		return 0, errors.New("missing Password")
	}

	return getUnseenCount(settings)
}

func getUnseenCount(settings Settings) (uint, error) {
	username := settings.Username
	password := settings.Password

	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return 0, fmt.Errorf("error while dialing the server: %w", err)
	}

	// Don't forget to logout
	defer func(c *client.Client) {
		err := c.Logout()
		if err != nil {
			log.Println("[gmail]", "unable to close imapClient", err)
		}
	}(c)

	if err := c.Login(username, password); err != nil {
		return 0, fmt.Errorf("error during login: %w", err)
	}

	const mailboxName = "INBOX"
	status, err := c.Status(mailboxName, []imap.StatusItem{imap.StatusUnseen})
	if err != nil {
		return 0, fmt.Errorf("unable to get status of %s: %w", mailboxName, err)
	}

	return uint(status.Unseen), nil
}

const RefreshInterval = time.Minute
