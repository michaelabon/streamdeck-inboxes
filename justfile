PLUGINS := "fastmail marvin ynab"

default:
    @just --list

install:
    npm install -g @elgato/cli
    git submodule update --init --recursive
    bun install

lint:
    bunx @biomejs/biome check --apply */*.sdPlugin/app.js

[macos]
link:
    for dir in {{ PLUGINS }}; do \
      ln -s \
        "{{ justfile_directory() }}/${dir}/ca.michaelabon.streamdeck-inboxes.${dir}.sdPlugin" \
        "$HOME/Library/Application Support/com.elgato.StreamDeck/Plugins" ; \
    done

[windows]
link:
    mklink /D "%AppData%\Elgato\StreamDeck\Plugins\ca.michaelabon.streamdeck-inboxes.fastmail.sdPlugin"   "{{ justfile_directory() }}/fastmail/ca.michaelabon.streamdeck-inboxes.fastmail.sdPlugin"
    mklink /D "%AppData%\Elgato\StreamDeck\Plugins\ca.michaelabon.streamdeck-inboxes.marvin.sdPlugin"   "{{ justfile_directory() }}/marvin/ca.michaelabon.streamdeck-inboxes.marvin.sdPlugin"
    mklink /D "%AppData%\Elgato\StreamDeck\Plugins\ca.michaelabon.streamdeck-inboxes.ynab.sdPlugin"   "{{ justfile_directory() }}/ynab/ca.michaelabon.streamdeck-inboxes.ynab.sdPlugin"

[macos]
debug:
    open "http://localhost:23654/"

[windows]
debug:
    start "" "http://localhost:23654/"


buildgo:
    go build -o ../ca.michaelabon.streamdeck-inboxes.sdPlugin/streamdeck-inboxes -C v2/go .

start:
    streamdeck restart ca.michaelabon.streamdeck-inboxes
