/// <reference path="sdk/js/action.js" />
/// <reference path="sdk/js/stream-deck.js" />

const myAction = new Action("ca.michaelabon.streamdeck-inboxes.ynab.action");

let interval = null;
let apiToken;
let xContext;

const MINUTES_PER_MILLISECOND = 1000 * 60;

const doUpdate = () => {
	if (!apiToken) {
		console.log("No api token");
		$SD.setTitle(xContext, "Setup");
	}
	getTransactionsCount(apiToken)
		.then((count) => {
			console.log(`SUCCESS on interval! “${count}”`);
			return $SD.setTitle(xContext, count);
		})
		.catch((err) => {
			$SD.logMessage(`EEEEE: ${err}`);
			$SD.setTitle(xContext, "!");
		});
};

const saveSettings = (payload) => {
	if (!payload) {
		return;
	}

	if (payload.hasOwnProperty("apiToken")) {
		apiToken = payload.apiToken;
	}

	if (
		payload.hasOwnProperty("settings") &&
		payload.settings.hasOwnProperty("apiToken")
	) {
		apiToken = payload.settings.apiToken;
	}
};

/**
 * The first event fired when Stream Deck starts
 */
$SD.onConnected(
	({ actionInfo, appInfo, connection, messageType, port, uuid }) => {
		console.log("Stream Deck connected!");

		$SD.getSettings();

		interval = setInterval(doUpdate, 2 * MINUTES_PER_MILLISECOND);
	},
);

myAction.onWillAppear(({ context, payload }) => {
	saveSettings(payload);
	xContext = context;
});

myAction.onWillDisappear((_x) => {
	if (interval) {
		clearInterval(interval);
	}
});

myAction.onKeyUp(({ action, context, device, event, payload }) => {
	doUpdate();

	$SD.send(context, Events.openUrl, {
		payload: {
			url: "https://app.ynab.com",
		},
	});
});

myAction.onDidReceiveSettings(({ context, payload }) => {
	console.log("Received settings", payload);
	saveSettings(payload);
});

async function getTransactionsCount(apiToken) {
	const budget = { id: "003b7b9b-22a9-4ec9-b943-201c7d014287" };

	const transactions = await (
		await fetch(
			`https://api.ynab.com/v1/budgets/${budget.id}/transactions?type=unapproved`,
			{
				headers: {
					Accept: "application/json",
					Authorization: `bearer ${apiToken}`,
				},
			},
		)
	).json();

	const filtered = transactions.data.transactions
		.filter((tx) => !tx.account_name.startsWith("[D]"))
		.filter((tx) => !tx.account_name.startsWith("[MD]"));

	return padRight(filtered.length, 7, " ");
}

function padRight(val, num, str) {
	let result = val.toString();
	const diff = num - result.length;
	for (let i = 0; i < diff; i++) {
		result = result + str;
	}
	return result;
}
