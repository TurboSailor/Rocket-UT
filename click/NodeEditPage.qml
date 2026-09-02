import QtQuick 2.7
import Ubuntu.Components 1.3

// Правка добавленного узла. Поля показываются по типу узла и собраны
// в секции-карточки: пустая секция целиком схлопывается в нулевую высоту.
// Конфиг AmneziaWG правится как файл, поэтому у awg-узла адрес только читается.
Page {
    id: page

    property var node: ({})
    // Тон плашки msg: обычно ошибка, для удачного замера — информационный.
    property string msgTone: "bad"

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
        page.msgTone = "bad"
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
        page.msgTone = "bad"
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

    // Замер задержки только этого узла: демон возвращает весь список,
    // из него берём свою запись и показываем её латентность в плашке.
    function test() {
        page.msgTone = "bad"
        msg.text = root.tr("testing…")
        root.api("/nodes/test?id=" + node.id, function(r, code) {
            if (!r || r.error) {
                msg.text = (r && r.error) || ("HTTP " + code)
                return
            }
            if (r && r.nodes) root.nodes = r.nodes
            var mine = null
            var list = r.nodes || []
            for (var i = 0; i < list.length; i++) {
                if (list[i].id === node.id) { mine = list[i]; break }
            }
            if (mine && mine.latency > 0) {
                page.msgTone = "info"
                msg.text = mine.latency + " ms"
                return
            }
            msg.text = root.tr("node unreachable")
        }, "POST")
    }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Edit node")

        RIconButton {
            name: "check"
            tint: pal.ok
            onClicked: page.save()
        }
        RIconButton {
            name: "trash"
            tint: pal.bad
            onClicked: page.remove()
        }
    }

    Flickable {
        id: flick
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        contentWidth: width
        contentHeight: col.height + units.gu(4)
        clip: true

        Column {
            id: col
            anchors {
                top: parent.top; topMargin: pal.pad
                left: parent.left; leftMargin: pal.pad
                right: parent.right; rightMargin: pal.pad
            }
            spacing: pal.gap

            RLabel {
                width: parent.width
                text: (node.type || "")
                      + (node.sub_id ? "  ·  " + root.tr("from subscription") : "")
            }

            RNote {
                width: parent.width
                tone: "warn"
                text: (node.sub_id !== undefined && node.sub_id !== "")
                      ? root.tr("editing detaches the node from its subscription") : ""
            }

            // --- основное ---
            RCard {
                width: parent.width
                height: visible ? basics.height + units.gu(3) : 0

                Column {
                    id: basics
                    anchors {
                        top: parent.top; topMargin: units.gu(1.5)
                        left: parent.left; leftMargin: units.gu(1.5)
                        right: parent.right; rightMargin: units.gu(1.5)
                    }
                    spacing: pal.gap

                    RLabel {
                        width: parent.width
                        section: true
                        text: root.tr("Basics")
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)

                        RLabel { width: parent.width; text: root.tr("Name") }
                        RField {
                            id: nameField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: !page.isAwg()

                        RLabel { width: parent.width; text: root.tr("Server") }
                        RField {
                            id: serverField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: !page.isAwg()

                        RLabel { width: parent.width; text: root.tr("Port") }
                        RField {
                            id: portField
                            width: parent.width
                            inputMethodHints: Qt.ImhDigitsOnly
                        }
                    }
                }
            }

            // --- доступ ---
            RCard {
                width: parent.width
                visible: page.needsCreds() || page.needsPassword()
                         || page.needsUUID() || page.needsMethod()
                height: visible ? access.height + units.gu(3) : 0

                Column {
                    id: access
                    anchors {
                        top: parent.top; topMargin: units.gu(1.5)
                        left: parent.left; leftMargin: units.gu(1.5)
                        right: parent.right; rightMargin: units.gu(1.5)
                    }
                    spacing: pal.gap

                    RLabel {
                        width: parent.width
                        section: true
                        text: root.tr("Access")
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsCreds()

                        RLabel { width: parent.width; text: root.tr("Username") }
                        RField {
                            id: userField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsPassword()

                        RLabel { width: parent.width; text: root.tr("Password") }
                        RField {
                            id: passField
                            width: parent.width
                            echoMode: TextInput.Normal
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsUUID()

                        RLabel { width: parent.width; text: "UUID" }
                        RField {
                            id: uuidField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsMethod()

                        RLabel { width: parent.width; text: root.tr("Method") }
                        RField {
                            id: methodField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
                        }
                    }
                }
            }

            // --- транспорт ---
            RCard {
                width: parent.width
                visible: page.needsTransport() || !page.isAwg()
                height: visible ? transport.height + units.gu(3) : 0

                Column {
                    id: transport
                    anchors {
                        top: parent.top; topMargin: units.gu(1.5)
                        left: parent.left; leftMargin: units.gu(1.5)
                        right: parent.right; rightMargin: units.gu(1.5)
                    }
                    spacing: pal.gap

                    RLabel {
                        width: parent.width
                        section: true
                        text: root.tr("Transport")
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsTransport()

                        RLabel { width: parent.width; text: "SNI" }
                        RField {
                            id: sniField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsTransport()

                        RLabel { width: parent.width; text: root.tr("Path") }
                        RField {
                            id: pathField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                        }
                    }

                    Column {
                        width: parent.width
                        spacing: units.gu(0.75)
                        visible: page.needsTransport()

                        RLabel { width: parent.width; text: root.tr("Host header") }
                        RField {
                            id: hostField
                            width: parent.width
                            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
                        }
                    }

                    Item {
                        width: parent.width
                        height: units.gu(3.4)
                        visible: !page.isAwg()

                        RSwitch {
                            id: tlsSwitch
                            anchors { left: parent.left; verticalCenter: parent.verticalCenter }
                        }
                        Text {
                            anchors {
                                left: tlsSwitch.right; leftMargin: units.gu(1.5)
                                right: parent.right
                                verticalCenter: parent.verticalCenter
                            }
                            text: "TLS"
                            color: pal.text
                            font.pixelSize: pal.fsBody
                            elide: Text.ElideRight
                        }
                    }

                    Item {
                        width: parent.width
                        height: units.gu(3.4)
                        visible: tlsSwitch.checked && !page.isAwg()

                        RSwitch {
                            id: insecureSwitch
                            anchors { left: parent.left; verticalCenter: parent.verticalCenter }
                        }
                        Text {
                            anchors {
                                left: insecureSwitch.right; leftMargin: units.gu(1.5)
                                right: parent.right
                                verticalCenter: parent.verticalCenter
                            }
                            text: root.tr("Skip cert check")
                            color: pal.text
                            font.pixelSize: pal.fsBody
                            elide: Text.ElideRight
                        }
                    }
                }
            }

            RNote {
                id: msg
                width: parent.width
                tone: page.msgTone
            }

            RButton {
                width: parent.width
                variant: "ghost"
                icon: "refresh"
                text: root.tr("Test")
                onClicked: page.test()
            }
            RButton {
                width: parent.width
                variant: "primary"
                text: root.tr("Save")
                onClicked: page.save()
            }
            RButton {
                width: parent.width
                variant: "danger"
                text: root.tr("Delete")
                onClicked: page.remove()
            }
        }
    }
}
