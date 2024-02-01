# Inboxes for Elgato Stream Deck

Stay on top of Inbox Zero with these Elgato Stream Deck actions.

## Keep on top of your unseen emails/tasks/purchases for:

- [Amazing Marvin][Marvin]
- [Fastmail][]
- [Gmail][]
- [You Need A Budget (YNAB)][YNAB]

## How to use it

Until I submit this plugin to the Elgato Marketplace, you must install it and build it manually.

You will need [Go](https://go.dev/dl/) to build this locally.
The file `go.mod` contains the minimum version of go required.

1. Clone the repository onto your Windows or macOS computer.
2. Install [just](https://github.com/casey/just) if you don't have it already.
   (`just` is an alternative to `make`).
3. Run `just build link`, to both build the plugin and create a symlink in your Elgato Plugins directory.
4. Restart the Elgato Stream Deck application on your computer.
5. In the Elgato Stream Deck application, on the right-hand side:
    1. Find the new *Inboxes* category.
    2. Drag-and-drop one of the actions into your Stream Deck grid.
6. Set the tokens or passwords on the Stream Deck keys. 
7. Enjoy that Inbox Zero.


## How do I contribute?

Pull requests are welcome.
For major changes, please open an issue first to discuss what you would like to change.

I have split the codebase into its two parts:

1. the actual Go code in `go/`
2. plugin-specific files are in `ca.michaelabon.streamdeck-inboxes.sdPlugin/`

Debugging is largely logfile based.
The plugin will write to the `ca.michaelabon.streamdeck-inboxes.sdPlugin/logs/` directory.

The [Stream Deck SDK](https://docs.elgato.com/sdk/) has documentation on how plugins work.
In short, the plugin and the configuration page (known as the property inspector) communicate with the Stream Deck over websockets.
They send and receive events.
The SDK documentation and building a Stream Deck plugin in general is far from the best developer experience.
If you have any questions about building a Stream Deck plugin, just open an issue in this repo.
I'll do my best to help.

Please make sure to update tests as appropriate.
You can run tests with `just test`.

## License

GNU General Public License v3.0, available at LICENSE


[Marvin]: https://amazingmarvin.com/
[Fastmail]: https://www.fastmail.com
[Gmail]: https://mail.google.com
[YNAB]: https://www.ynab.com/
