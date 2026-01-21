package gmail

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// DefaultMailbox is the default Gmail mailbox to monitor.
const DefaultMailbox = "INBOX"

// mailboxChannelBuffer is the buffer size for the mailbox listing channel.
const mailboxChannelBuffer = 100

type Settings struct {
	Username string
	Password string
	Label    string // Gmail label/mailbox to monitor (default: "INBOX")
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

	// Default to INBOX if no label specified
	mailboxName := settings.Label
	if mailboxName == "" {
		mailboxName = DefaultMailbox
	}

	status, err := c.Status(mailboxName, []imap.StatusItem{imap.StatusUnseen})
	if err != nil {
		return 0, fmt.Errorf("unable to get status of %s: %w", mailboxName, err)
	}

	return uint(status.Unseen), nil
}

// FetchLabels returns all available Gmail mailbox names (labels).
func FetchLabels(settings Settings) ([]string, error) {
	if settings.Username == "" {
		return nil, errors.New("missing Username")
	}
	if settings.Password == "" {
		return nil, errors.New("missing Password")
	}

	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		return nil, fmt.Errorf("error while dialing the server: %w", err)
	}

	defer func(c *client.Client) {
		err := c.Logout()
		if err != nil {
			log.Println("[gmail]", "unable to close imapClient", err)
		}
	}(c)

	if err := c.Login(settings.Username, settings.Password); err != nil {
		return nil, fmt.Errorf("error during login: %w", err)
	}

	// List all mailboxes
	mailboxes := make(chan *imap.MailboxInfo, mailboxChannelBuffer)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	var labels []string
	for m := range mailboxes {
		labels = append(labels, m.Name)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("error listing mailboxes: %w", err)
	}

	// Sort labels alphabetically, but put INBOX first
	sort.Strings(labels)
	for i, label := range labels {
		if label == DefaultMailbox && i > 0 {
			labels = append([]string{DefaultMailbox}, append(labels[:i], labels[i+1:]...)...)

			break
		}
	}

	return labels, nil
}

const RefreshInterval = time.Minute
