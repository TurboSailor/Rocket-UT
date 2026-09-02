import QtQuick 2.7
import Ubuntu.Components 1.3

// Конфиги: добавление по ссылке или из файла и список конфигов.
// Активация и удаление разнесены по отдельным кнопкам строки.
Page {
    id: page
    property string activeConf: ""

    onVisibleChanged: if (visible) reload()

    function reload() {
        root.api("/confs", function(r) {
            if (!r) return
            root.confs = r.confs || []
            page.activeConf = r.active || ""
        })
    }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Configs")
    }

    // --- добавление конфига ---
    RCard {
        id: formCard
        anchors {
            top: hdr.bottom; topMargin: pal.pad
            left: parent.left; leftMargin: pal.pad
            right: parent.right; rightMargin: pal.pad
        }
        height: form.height + units.gu(3)

        Column {
            id: form
            anchors {
                left: parent.left; leftMargin: units.gu(1.5)
                right: parent.right; rightMargin: units.gu(1.5)
                verticalCenter: parent.verticalCenter
            }
            spacing: pal.gap

            RField {
                id: nameField
                width: parent.width
                placeholderText: root.tr("Config name")
                text: "shadowrocket"
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }
            RField {
                id: urlField
                width: parent.width
                placeholderText: root.tr("URL")
                inputMethodHints: Qt.ImhUrlCharactersOnly | Qt.ImhNoPredictiveText
            }
            RButton {
                width: parent.width
                variant: "primary"
                text: root.tr("Add from URL")
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
            RButton {
                width: parent.width
                variant: "ghost"
                icon: "folder"
                text: root.tr("Open file")
                onClicked: { root.pendingFile = "conf"; stack.push(filePage) }
            }
            RNote {
                id: msg
                width: parent.width
                tone: "warn"
            }
        }
    }

    // --- список конфигов ---
    ListView {
        id: list
        anchors {
            top: formCard.bottom
            bottom: parent.bottom
            left: parent.left
            right: parent.right
        }
        model: root.confs || []
        topMargin: pal.gap
        bottomMargin: pal.pad
        spacing: units.gu(1)
        clip: true

        delegate: RRow {
            x: pal.pad
            width: list.width - 2 * pal.pad
            icon: "file"
            tint: modelData === page.activeConf ? pal.accent : pal.dim
            title: modelData
            subtitle: modelData === page.activeConf ? root.tr("active") : ""
            active: modelData === page.activeConf
            chevron: true
            onClicked: {
                confEdit.confName = modelData
                stack.push(confEdit)
            }

            RIconButton {
                visible: modelData !== page.activeConf
                name: "play"
                tint: pal.ok
                onClicked: {
                    root.busy = root.tr("working…")
                    root.api("/conf/select?name=" + encodeURIComponent(modelData),
                             function(r) { root.busy = ""; root.absorb(r); reload() }, "POST")
                }
            }
            RIconButton {
                name: "trash"
                tint: pal.bad
                onClicked: root.api("/conf/delete?name=" + encodeURIComponent(modelData),
                                    function(r) { reload(); root.refresh() }, "POST")
            }
        }
    }

    REmpty {
        anchors.centerIn: list
        visible: (root.confs || []).length === 0
        icon: "file"
        text: root.tr("no configs yet")
    }
}
