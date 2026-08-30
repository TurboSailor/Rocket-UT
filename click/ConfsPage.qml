import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.ListItems 1.3 as ListItem

Page {
    id: page
    property string activeConf: ""

    header: PageHeader {
        id: hdr
        title: root.tr("Configs")
    }
    onVisibleChanged: if (visible) reload()

    function reload() {
        root.api("/confs", function(r) {
            if (!r) return
            root.confs = r.confs || []
            page.activeConf = r.active || ""
        })
    }

    Column {
        id: form
        anchors { top: hdr.bottom; left: parent.left; right: parent.right; margins: units.gu(2) }
        spacing: units.gu(1)

        TextField {
            id: nameField
            width: parent.width
            placeholderText: root.tr("Config name")
            text: "shadowrocket"
            inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
        }
        TextField {
            id: urlField
            width: parent.width
            placeholderText: root.tr("URL")
            inputMethodHints: Qt.ImhUrlCharactersOnly | Qt.ImhNoPredictiveText
        }
        Button {
            width: parent.width
            text: root.tr("Add from URL")
            color: UbuntuColors.green
            enabled: urlField.text.trim() !== "" && nameField.text.trim() !== ""
            onClicked: {
                msg.text = root.tr("working…")
                root.api("/conf/fromurl?name=" + encodeURIComponent(nameField.text.trim())
                         + "&url=" + encodeURIComponent(urlField.text.trim()),
                    function(r, code) {
                        if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
                        msg.text = r.rules + " " + root.tr("rules") + ", "
                                 + r.proxies + " " + root.tr("proxies")
                                 + (r.skipped && r.skipped.length
                                    ? ", " + r.skipped.length + " " + root.tr("unsupported lines") : "")
                        urlField.text = ""
                        reload(); root.refresh()
                    }, "POST")
            }
        }
        Button {
            width: parent.width
            text: root.tr("Open file")
            onClicked: { root.pendingFile = "conf"; stack.push(filePage) }
        }
        Label {
            id: msg
            width: parent.width
            wrapMode: Text.Wrap
            fontSize: "small"
            color: UbuntuColors.orange
        }
    }

    Label {
        anchors.centerIn: parent
        visible: (root.confs || []).length === 0
        text: root.tr("no configs yet")
        color: UbuntuColors.graphite
    }

    ListView {
        anchors { top: form.bottom; topMargin: units.gu(1); bottom: parent.bottom
                  left: parent.left; right: parent.right }
        model: root.confs || []
        clip: true
        delegate: ListItem.Standard {
            text: modelData + (modelData === page.activeConf ? "  ●  " + root.tr("active") : "")
            selected: modelData === page.activeConf
            onClicked: {
                confEdit.confName = modelData
                stack.push(confEdit)
            }
            control: Button {
                text: modelData === page.activeConf ? "✕" : "▶"
                width: units.gu(4.5)
                onClicked: {
                    if (modelData === page.activeConf) {
                        root.api("/conf/delete?name=" + encodeURIComponent(modelData),
                                 function(r) { reload(); root.refresh() }, "POST")
                    } else {
                        root.busy = root.tr("working…")
                        root.api("/conf/select?name=" + encodeURIComponent(modelData),
                                 function(r) { root.busy = ""; root.absorb(r); reload() }, "POST")
                    }
                }
            }
        }
    }
}
