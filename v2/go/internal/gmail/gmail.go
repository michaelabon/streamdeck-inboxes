package gmail

import (
	"context"
	"log"
	"strconv"
	"time"

	"ca.michaelabon.inboxes/internal/display"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/samwho/streamdeck"
)

type Settings struct {
	Username string
	Password string
}

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

	err = client.SetTitle(ctx, display.PadRight(strconv.Itoa(int(unseenCount))), streamdeck.HardwareAndSoftware)
	if err != nil {
		log.Println("error while setting icon title with unseen count", err)
		return err
	}

	return nil
}

func getUnseenCount(settings *Settings) (uint32, error) {
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

	unseen := status.Unseen

	return unseen, nil
}

const RefreshInterval = time.Minute
