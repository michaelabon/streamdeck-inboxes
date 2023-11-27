/// <reference path="sdk/js/action.js" />
/// <reference path="sdk/js/stream-deck.js" />

const myAction = new Action("ca.michaelabon.streamdeck-inboxes.marvin.action")

let interval = null

let marvinServer
let marvinDatabase
let marvinUser
let marvinPassword
let marvinSettings

let xContext

const MINUTES_PER_MILLISECOND = 1000 * 60

const doUpdate = () => {
	getInboxCount()
		.then((count) => {
			console.log("SUCCESS!", count)
			return $SD.setTitle(xContext, count)
		})
		.catch((err) => $SD.logMessage(`EEEEE: ${err}`))
}

/**
 * The first event fired when Stream Deck starts
 */
$SD.onConnected(
	({ actionInfo, appInfo, connection, messageType, port, uuid }) => {
		console.log("Stream Deck connected!")

		$SD.getSettings()

		interval = setInterval(doUpdate, 2 * MINUTES_PER_MILLISECOND)
	},
)

myAction.onWillAppear(({ context, payload }) => {
	xContext = context

	console.log("Will appear", payload)
	$SD.setTitle(context, padRight("?", 7, " "))

	saveSettings(payload)

	doUpdate()
})

myAction.onWillDisappear((_x) => {
	if (interval) {
		clearInterval(interval)
	}
})

myAction.onKeyUp(({ action, context, device, event, payload }) => {
	saveSettings(payload)

	doUpdate()

	$SD.send(context, Events.openUrl, {
		payload: {
			url: "https://app.amazingmarvin.com",
		},
	})
})

function saveSettings(payload) {
	if (!payload) {
		return false
	}

	if (!payload.settings) {
		return false
	}

	const { syncServer, syncDatabase, syncUser, syncPassword } = payload.settings

	if (!syncServer || !syncDatabase || !syncUser || !syncPassword) {
		return false
	}

	marvinServer = syncServer
	marvinDatabase = syncDatabase
	marvinUser = syncUser
	marvinPassword = syncPassword

	return true
}

myAction.onDidReceiveSettings(({ context, payload }) => {
	console.log("Received settings", payload)

	if (!saveSettings(payload)) {
		return false
	}

	doUpdate()
})

async function thisFetch(url) {
	console.log("this fetch!")
	const base64 = btoa(`${marvinUser}:${marvinPassword}`)
	const init = {
		method: "GET",
		headers: {
			Accept: "application/json",
			Authorization: `Basic ${base64}`,
		},
	}
	console.log("init", init)
	const response = await fetch(url, init)

	console.log("Raw response", response)

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

async function getInboxCount() {
	const childrenUrl = new URL(`${marvinDatabase}/_all_docs`, marvinServer)
	childrenUrl.searchParams.set("include_docs", "true")

	const children = await thisFetch(childrenUrl)

	const filtered = children.rows
		.map((task) => task.doc)
		.filter((task) => task.title)
		.filter((task) => task.db === "Tasks")
		.filter((task) => task.parentId === "unassigned")
		.filter((task) => !task.done)
		.filter((task) => !task.recurring)

	return padRight(filtered.length, 7, " ")
}

function padRight(val, num, str) {
	let result = val.toString()
	const diff = num - result.length
	for (let i = 0; i < diff; i++) {
		result = result + str
	}
	return result
}
