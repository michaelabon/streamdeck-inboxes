/// <reference path="./sdk/js/property-inspector.js" />
/// <reference path="./sdk/js/utils.js" />

$PI.onConnected((jsn) => {
    const form = document.querySelector('#property-inspector');
    const {actionInfo, appInfo, connection, messageType, port, uuid} = jsn;
    const {payload, context} = actionInfo;
    const {settings} = payload;

    Utils.setFormValue(settings, form);

    form.addEventListener(
        'input',
        Utils.debounce(150, () => {
            const value = Utils.getFormValue(form);
            $PI.setSettings(value);
        })
    );

    window.onGetSettingsClick = (url) => {
        $PI.send(this.UUID, "openUrl", {payload: {url}})
    }
});

$PI.onDidReceiveGlobalSettings(({payload}) => {
    console.log('onDidReceiveGlobalSettings', payload);
})
