htmx.defineExtension('json-enc', {
    onEvent: function (name, evt) {
        if (name === "htmx:configRequest") {
            evt.detail.headers['Content-Type'] = "application/json";
        }
    },
    encodeParameters: function(helper, elt, parameters) {
        helper.getHeaders()['Content-Type'] = "application/json";
        return JSON.stringify(parameters);
    }
});
