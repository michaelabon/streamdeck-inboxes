/// <reference path="sdk/js/action.js" />
/// <reference path="sdk/js/stream-deck.js" />

const myAction = new Action("ca.michaelabon.streamdeck-inboxes.fastmail.action")

let interval = null

let fastmailApiToken

let xContext

let accountId

const MINUTES_PER_MILLISECOND = 1000 * 60

const doUpdate = () => {
	getInboxCount()
		.then((count) => {
			console.log("SUCCESS!", count)
			$SD.setTitle(xContext, count)
		})
		.catch((err) => {
			$SD.logMessage(`EEEEE: ${err}`)
			$SD.setTitle(xContext, "!")
			$SD.showAlert(xContext)
		})
}

/**
 * The first event fired when Stream Deck starts
 *
 * {actionInfo, appInfo, connection, messageType, port, uuid} = onConnectedOpts
 */
$SD.onConnected((onConnectedOpts) => {
	console.log("Stream Deck connected!", onConnectedOpts)

	$SD.getSettings()

	interval = setInterval(doUpdate, MINUTES_PER_MILLISECOND)
})

myAction.onWillAppear(({ context, payload }) => {
	xContext = context

	console.log("Will appear", payload)

	if (payload.settings) {
		fastmailApiToken = payload.settings.fastmailApiToken
	}

	$SD.setTitle(context, padRight("?", 7, " "))
	doUpdate()
})

myAction.onWillDisappear((_x) => {
	if (interval) {
		clearInterval(interval)
	}
})

myAction.onKeyUp(({ action, context, device, event, payload }) => {
	xContent = context

	doUpdate()

	$SD.send(context, Events.openUrl, {
		payload: {
			url: "https://app.fastmail.com/mail/Inbox",
		},
	})
})

myAction.onDidReceiveSettings(({ context, payload }) => {
	console.log("Received settings", payload)

	xContext = context

	if (payload?.settings) {
		fastmailApiToken = payload.settings.fastmailApiToken
		accountId = undefined
	}

	doUpdate()
})

async function thisFetch(url, body = null) {
	console.log("this fetch!")

	const init = {
		method: "GET",
		headers: {
			Authorization: `Bearer ${fastmailApiToken}`,
			Accept: "application/json",
		},
	}

	if (body) {
		init.method = "POST"
		init.body = JSON.stringify(body)
		init.headers["Content-Type"] = "application/json"
	}

	console.log("init", init)
	const response = await fetch(url, init)

	if (response.status >= 400) {
		console.error("ERROR RESPONSE")
		console.error("Request:", url)
		console.error("Request body:", init)
		console.error("Response headers", response.headers)

		const text = await response.text()
		console.error("Response body:", text)

		throw new Error(text)
	}
	return response.json()
}

const BASE_URL = "https://api.fastmail.com/jmap/"
const SESSION_URL = new URL("session", BASE_URL)
const API_URL = new URL("api", BASE_URL)

async function getInboxCount() {
	if (!accountId) {
		const session = await thisFetch(SESSION_URL)

		console.log("SESSION RESPONSE", session)

		accountId = session.primaryAccounts["urn:ietf:params:jmap:mail"]

		if (!accountId) {
			throw new Error(
				`Unable to retrieve accountId from session response: ${JSON.stringify(
					session,
				)}`,
			)
		}
	}

	const body = {
		using: ["urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"],
		methodCalls: [
			[
				"Mailbox/get",
				{
					accountId: accountId,
					ids: null,
				},
				"0",
			],
		],
	}

	const accountDetails = await thisFetch(API_URL, body)

	console.log("Account details", accountDetails)

	const methodResponse = accountDetails.methodResponses.find(
		(responseArray) =>
			responseArray[0] === "Mailbox/get" && responseArray[2] === "0",
	)

	if (!methodResponse) {
		throw new Error(
			`Unable to retrieve the Mailbox/get methodResponse from account response: ${JSON.stringify(
				accountDetails,
			)}`,
		)
	}

	const folders = methodResponse[1].list
	const inbox = folders.find((folder) => folder.role === "inbox")
	const totalEmails = inbox.totalEmails
	const unreadEmails = inbox.unreadEmails

	return padRight(`${unreadEmails}/${totalEmails}`, 7, " ")
}

function padRight(val, num, str) {
	let result = val.toString()
	const diff = num - result.length
	for (let i = 0; i < diff; i++) {
		result = result + str
	}
	return result
}
