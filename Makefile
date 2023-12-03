UNAME := $(shell uname)
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
PLUGINS := fastmail marvin ynab
PERCENT := %

.PHONY: install link lint

install:
	git submodule update --init --recursive
	bun install

lint:
	bunx @biomejs/biome check --apply fastmail/ca.michaelabon.streamdeck-inboxes.fastmail.sdPlugin/app.js marvin/ca.michaelabon.streamdeck-inboxes.marvin.sdPlugin/app.js ynab/ca.michaelabon.streamdeck-inboxes.ynab.sdPlugin/app.js


link:
ifeq ($(UNAME), Darwin)
	for dir in $(PLUGINS); do \
		ln -s $(ROOT_DIR)/$$dir/ca.michaelabon.streamdeck-inboxes.$$dir.sdPlugin   "${HOME}/Library/Application Support/com.elgato.StreamDeck/Plugins"; \
	done
else
	mklink /D "$(PERCENT)AppData$(PERCENT)\Elgato\StreamDeck\Plugins\ca.michaelabon.streamdeck-inboxes.fastmail.sdPlugin"   "${PERCENT}cd${PERCENT}/fastmail/ca.michaelabon.streamdeck-inboxes.fastmail.sdPlugin"
	mklink /D "$(PERCENT)AppData$(PERCENT)\Elgato\StreamDeck\Plugins\ca.michaelabon.streamdeck-inboxes.marvin.sdPlugin"   "${PERCENT}cd${PERCENT}/marvin/ca.michaelabon.streamdeck-inboxes.marvin.sdPlugin"
	mklink /D "$(PERCENT)AppData$(PERCENT)\Elgato\StreamDeck\Plugins\ca.michaelabon.streamdeck-inboxes.ynab.sdPlugin"   "${PERCENT}cd${PERCENT}/ynab/ca.michaelabon.streamdeck-inboxes.ynab.sdPlugin"
endif

