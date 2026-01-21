/// <reference path="./sdk/js/property-inspector.js" />
/// <reference path="./sdk/js/utils.js" />

const ACTION_UUID = 'ca.michaelabon.streamdeck-inboxes.gmail.action';

$PI.onConnected((jsn) => {
    const form = document.querySelector('#property-inspector');
    const {actionInfo, appInfo, connection, messageType, port, uuid} = jsn;
    const {payload, context} = actionInfo;
    const {settings} = payload;

    Utils.setFormValue(settings, form);

    const labelSelect = document.getElementById('label-select');
    const labelStatus = document.getElementById('label-status');
    const labelStatusText = document.getElementById('label-status-text');

    // Set initial label value if exists and credentials are present
    if (settings.label && settings.username && settings.password) {
        labelSelect.innerHTML = '';
        const option = document.createElement('option');
        option.value = settings.label;
        option.text = settings.label;
        option.selected = true;
        labelSelect.appendChild(option);
    }

    // Function to request labels from plugin
    function fetchLabels() {
        const formValues = Utils.getFormValue(form);
        if (formValues.username && formValues.password) {
            labelSelect.disabled = true;
            labelSelect.innerHTML = '<option value="">Loading...</option>';
            labelStatus.style.display = 'none';

            $PI.sendToPlugin({
                action: 'fetchLabels',
                settings: formValues
            });
        } else {
            labelSelect.disabled = true;
            labelSelect.innerHTML = '<option value="">Enter credentials first</option>';
            labelStatus.style.display = 'none';
        }
    }

    // Listen for responses from the plugin
    $PI.onSendToPropertyInspector(ACTION_UUID, (data) => {
        const {payload} = data;

        if (payload.action === 'fetchLabels') {
            if (payload.error) {
                labelSelect.disabled = true;
                labelSelect.innerHTML = '<option value="">Failed to load</option>';
                labelStatus.style.display = 'block';
                labelStatusText.textContent = payload.error;
            } else {
                const currentValue = settings.label || 'INBOX';
                labelSelect.innerHTML = '';

                payload.labels.forEach(label => {
                    const option = document.createElement('option');
                    option.value = label;
                    option.text = label;
                    if (label === currentValue) {
                        option.selected = true;
                    }
                    labelSelect.appendChild(option);
                });

                labelSelect.disabled = false;
                labelStatus.style.display = 'none';
            }
        }
    });

    // Fetch labels on credential change (debounced)
    const credentialInputs = form.querySelectorAll('input[name="username"], input[name="password"]');
    credentialInputs.forEach(input => {
        input.addEventListener('input', Utils.debounce(500, () => {
            fetchLabels();
        }));
    });

    // Standard form change handler
    form.addEventListener(
        'input',
        Utils.debounce(150, () => {
            const value = Utils.getFormValue(form);
            $PI.setSettings(value);
        })
    );

    // Fetch labels on initial load if credentials exist
    if (settings.username && settings.password) {
        fetchLabels();
    }

    window.onGetSettingsClick = (url) => {
        $PI.send(this.UUID, "openUrl", {payload: {url}});
    };
});

$PI.onDidReceiveGlobalSettings(({payload}) => {
    console.log('onDidReceiveGlobalSettings', payload);
});
