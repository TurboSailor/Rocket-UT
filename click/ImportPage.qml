import QtQuick 2.7
import Ubuntu.Components 1.3

Page {
    id: page

    header: PageHeader {
        id: hdr
        title: root.tr("Import")
        flickable: flick
    }

    // setText вызывается из FilePage после чтения файла.
    function setText(t) { area.text = t }

    Flickable {
        id: flick
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        contentWidth: width
        contentHeight: col.height + units.gu(4)
        clip: true

        Column {
            id: col
            width: parent.width - units.gu(4)
            anchors { top: parent.top; topMargin: units.gu(2); horizontalCenter: parent.horizontalCenter }
            spacing: units.gu(1)

            Label {
                width: parent.width
                wrapMode: Text.Wrap
                fontSize: "small"
                text: root.tr("Import from file or paste text below")
            }
            Label {
                width: parent.width
                wrapMode: Text.Wrap
                fontSize: "x-small"
                color: UbuntuColors.graphite
                text: root.tr("Supported: vless, vmess, ss, trojan, socks5, http, ssh URIs, sub:// links, Shadowrocket .conf, AmneziaWG .conf")
            }

            Button {
                width: parent.width
                text: root.tr("Scan QR")
                color: UbuntuColors.orange
                onClicked: stack.push(scanPage)
            }
            Button {
                width: parent.width
                text: root.tr("QR from image")
                onClicked: { root.pendingFile = "qr"; stack.push(filePage) }
            }
            Button {
                width: parent.width
                text: root.tr("Open file")
                onClicked: { root.pendingFile = "import"; stack.push(filePage) }
            }
            Label {
                width: parent.width
                text: root.tr("AmneziaWG name")
                fontSize: "small"
            }
            TextField {
                id: nameField
                width: parent.width
                text: "awg"
                placeholderText: root.tr("AmneziaWG name")
                maximumLength: 32
                inputMethodHints: Qt.ImhPreferLowercase | Qt.ImhNoPredictiveText
            }

            TextArea {
                id: area
                width: parent.width
                height: units.gu(28)
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                placeholderText: root.tr("Paste links or config")
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
                text: root.tr("Import")
                color: UbuntuColors.green
                enabled: area.text.trim() !== ""
                onClicked: {
                    msg.text = root.tr("working…")
                    root.api("/nodes/import?name=" + encodeURIComponent(nameField.text.trim() || "awg"),
                        function(r, code) {
                            if (!r || r.error) {
                                msg.text = (r && r.error) || ("HTTP " + code)
                                return
                            }
                            msg.text = ""
                            root.absorb(r)
                            root.refresh()
                            area.text = ""
                            stack.pop()
                        }, "POST", area.text)
                }
            }
        }
    }
}
