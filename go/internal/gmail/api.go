package gmail

import (
	"errors"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type Settings struct {
	Username string
	Password string
}

func FetchUnseenCount(settings *Settings) (uint, error) {
	if settings.Username == "" {
		return 0, errors.New("missing Username")
	}
	if settings.Password == "" {
		return 0, errors.New("missing Password")
	}

	return getUnseenCount(settings)
}

func getUnseenCount(settings *Settings) (uint, error) {
	username := settings.Username
	password := settings.Password

	log.Println("Connecting to server...")

	// Connect to server
	c, err := client.DialTLS("imap.gmail.com:993", nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected")

	// Don't forget to logout
	defer func(c *client.Client) {
		err := c.Logout()
		if err != nil {
			log.Println("unable to close imapClient", err)
		}
	}(c)

	// Login
	if err := c.Login(username, password); err != nil {
		log.Fatal(err)
	}

	log.Println("Logged in")

	status, err := c.Status("INBOX", []imap.StatusItem{imap.StatusUnseen})
	if err != nil {
		log.Fatal(err)
	}

	return uint(status.Unseen), nil
}

const RefreshInterval = time.Minute
