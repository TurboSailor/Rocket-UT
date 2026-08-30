import QtQuick 2.7
import Ubuntu.Components 1.3

// Правка добавленного узла. Поля показываются по типу узла.
// Конфиг AmneziaWG правится как файл, поэтому у awg-узла адрес только читается.
Page {
    id: page

    property var node: ({})

    header: PageHeader {
        id: hdr
        title: root.tr("Edit node")
        flickable: flick
        trailingActionBar.actions: [
            Action {
                iconName: "save"
                text: root.tr("Save")
                onTriggered: page.save()
            },
            Action {
                iconName: "delete"
                text: root.tr("Delete")
                onTriggered: page.remove()
            }
        ]
    }

    onVisibleChanged: if (visible) page.load()

    function load() {
        msg.text = ""
        nameField.text = node.name || ""
        serverField.text = node.server || ""
        portField.text = node.port ? String(node.port) : ""
        userField.text = node.user || ""
        passField.text = node.password || ""
        uuidField.text = node.uuid || ""
        methodField.text = node.method || ""
        sniField.text = node.sni || ""
        pathField.text = node.path || ""
        hostField.text = node.host || ""
        tlsSwitch.checked = node.tls === true
        insecureSwitch.checked = node.insecure === true
    }

    function isAwg() { return node.type === "awg" }

    // needs: какие поля осмысленны для типа узла.
    function needsCreds() {
        return node.type === "socks5" || node.type === "http" || node.type === "ssh"
    }
    function needsUUID() { return node.type === "vless" || node.type === "vmess" }
    function needsPassword() {
        return node.type === "ss" || node.type === "trojan" || page.needsCreds()
    }
    function needsMethod() { return node.type === "ss" || node.type === "vmess" }
    function needsTransport() {
        return node.type === "vless" || node.type === "vmess" || node.type === "trojan"
    }

    function save() {
        var payload = {
            name: nameField.text.trim(),
            server: serverField.text.trim(),
            port: parseInt(portField.text.trim(), 10) || 0,
            user: userField.text,
            password: passField.text,
            uuid: uuidField.text.trim(),
            method: methodField.text.trim(),
            sni: sniField.text.trim(),
            path: pathField.text,
            host: hostField.text.trim(),
            tls: tlsSwitch.checked,
            insecure: insecureSwitch.checked
        }
        msg.text = root.tr("working…")
        root.api("/nodes/update?id=" + node.id, function(r, code) {
            if (!r || r.error) {
                msg.text = (r && r.error) || ("HTTP " + code)
                return
            }
            msg.text = ""
            root.absorb(r)
            root.refresh()
            stack.pop()
        }, "POST", JSON.stringify(payload))
    }

    function remove() {
        msg.text = root.tr("working…")
        root.api("/nodes/delete?id=" + node.id, function(r, code) {
            if (!r || r.error) {
                msg.text = (r && r.error) || ("HTTP " + code)
                return
            }
            msg.text = ""
            root.absorb(r)
            root.refresh()
            stack.pop()
        }, "POST")
    }

    Flickable {
        id: flick
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        contentWidth: width
        contentHeight: col.height + units.gu(4)
        clip: true

        Column {
            id: col
            width: parent.width - units.gu(4)
            anchors { top: parent.top; topMargin: units.gu(1); horizontalCenter: parent.horizontalCenter }
            spacing: units.gu(1)

            Label {
                width: parent.width
                fontSize: "small"
                color: UbuntuColors.graphite
                text: (node.type || "") + (node.sub_id ? "  ·  " + root.tr("from subscription") : "")
            }
            Label {
                width: parent.width
                wrapMode: Text.Wrap
                fontSize: "x-small"
                color: UbuntuColors.orange
                visible: node.sub_id !== undefined && node.sub_id !== ""
                text: root.tr("editing detaches the node from its subscription")
            }

            Label { text: root.tr("Name"); fontSize: "small" }
            TextField {
                id: nameField
                width: parent.width
                inputMethodHints: Qt.ImhNoPredictiveText
            }

            Label { text: root.tr("Server"); fontSize: "small"; visible: !page.isAwg() }
            TextField {
                id: serverField
                width: parent.width
                visible: !page.isAwg()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }

            Label { text: root.tr("Port"); fontSize: "small"; visible: !page.isAwg() }
            TextField {
                id: portField
                width: parent.width
                visible: !page.isAwg()
                inputMethodHints: Qt.ImhDigitsOnly
            }

            Label { text: root.tr("Username"); fontSize: "small"; visible: page.needsCreds() }
            TextField {
                id: userField
                width: parent.width
                visible: page.needsCreds()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
            }

            Label { text: root.tr("Password"); fontSize: "small"; visible: page.needsPassword() }
            TextField {
                id: passField
                width: parent.width
                visible: page.needsPassword()
                echoMode: TextInput.Normal
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
            }

            Label { text: "UUID"; fontSize: "small"; visible: page.needsUUID() }
            TextField {
                id: uuidField
                width: parent.width
                visible: page.needsUUID()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
            }

            Label { text: root.tr("Method"); fontSize: "small"; visible: page.needsMethod() }
            TextField {
                id: methodField
                width: parent.width
                visible: page.needsMethod()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }

            Label { text: "SNI"; fontSize: "small"; visible: page.needsTransport() }
            TextField {
                id: sniField
                width: parent.width
                visible: page.needsTransport()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }

            Label { text: root.tr("Path"); fontSize: "small"; visible: page.needsTransport() }
            TextField {
                id: pathField
                width: parent.width
                visible: page.needsTransport()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
            }

            Label { text: root.tr("Host header"); fontSize: "small"; visible: page.needsTransport() }
            TextField {
                id: hostField
                width: parent.width
                visible: page.needsTransport()
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }

            Row {
                width: parent.width
                spacing: units.gu(1)
                visible: !page.isAwg()
                Switch { id: tlsSwitch }
                Label { text: "TLS"; anchors.verticalCenter: parent.verticalCenter }
            }
            Row {
                width: parent.width
                spacing: units.gu(1)
                visible: tlsSwitch.checked && !page.isAwg()
                Switch { id: insecureSwitch }
                Label {
                    text: root.tr("Skip cert check")
                    anchors.verticalCenter: parent.verticalCenter
                }
            }

            Label {
                id: msg
                width: parent.width
                wrapMode: Text.Wrap
                fontSize: "small"
                color: UbuntuColors.red
            }

            Button {
                width: parent.width
                text: root.tr("Save")
                color: UbuntuColors.green
                onClicked: page.save()
            }
            Button {
                width: parent.width
                text: root.tr("Delete")
                color: UbuntuColors.red
                onClicked: page.remove()
            }
        }
    }
}
