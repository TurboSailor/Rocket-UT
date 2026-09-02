import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.Popups 1.3 as Popups

// Журнал соединений: свежие сверху, тап по строке заводит правило.
Page {
    id: page

    ListModel { id: entries }

    function reload() {
        root.api("/log", function(r) {
            if (!r || !r.entries) return
            entries.clear()
            // Свежие сверху: журнал читают с конца.
            for (var i = r.entries.length - 1; i >= 0; i--)
                entries.append(r.entries[i])
        })
    }

    onVisibleChanged: if (visible) reload()

    Timer {
        interval: 1000
        running: page.visible
        repeat: true
        onTriggered: page.reload()
    }

    function policyColor(p) {
        if (p === "PROXY") return pal.accent
        if (p === "REJECT") return pal.bad
        return pal.ok
    }

    // Значок строки повторяет смысл политики: прокси, запрет, напрямую.
    function policyIcon(p) {
        if (p === "PROXY") return "globe"
        if (p === "REJECT") return "close"
        return "check"
    }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Log")

        RIconButton {
            name: "refresh"
            tint: pal.accent
            onClicked: page.reload()
        }
    }

    REmpty {
        anchors.centerIn: parent
        visible: entries.count === 0
        icon: "clock"
        text: root.tr("no traffic yet")
    }

    ListView {
        id: list
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        model: entries
        topMargin: pal.pad
        bottomMargin: pal.pad
        spacing: units.gu(1)
        clip: true

        delegate: RRow {
            x: pal.pad
            width: list.width - 2 * pal.pad
            icon: page.policyIcon(model.policy)
            tint: page.policyColor(model.policy)
            title: model.host || model.ip
            subtitle: root.fmtTime(model.time) + " · " + model.net + ":" + model.port
                      + " → " + model.policy
            value: "↓" + root.fmtBytes(model.down) + " ↑" + root.fmtBytes(model.up)
            valueColor: pal.dim
            chevron: true

            // Живое соединение помечается залитой точкой.
            RIcon {
                visible: model.open
                name: "dot"
                tint: pal.ok
                size: units.gu(1.2)
            }

            onClicked: {
                var host = model.host || model.ip
                Popups.PopupUtils.open(ruleDialog, page, {targetHost: host, targetIP: model.ip})
            }
        }
    }

    Component {
        id: ruleDialog
        Popups.Dialog {
            id: dlg
            property string targetHost: ""
            property string targetIP: ""
            property var types: ["DOMAIN", "DOMAIN-SUFFIX", "IP-CIDR"]
            property var policies: ["PROXY", "DIRECT", "REJECT"]
            property int typeIndex: 1
            property int policyIndex: 0

            title: root.tr("Add rule")

            // suggest подставляет значение под выбранный тип правила.
            function suggest(idx) {
                if (idx === 2)
                    return dlg.targetIP ? dlg.targetIP + "/32" : ""
                if (idx === 1) {
                    var p = dlg.targetHost.split(".")
                    return p.length > 2 ? p.slice(p.length - 2).join(".") : dlg.targetHost
                }
                return dlg.targetHost
            }

            Item {
                id: body
                width: parent ? parent.width : units.gu(34)
                height: form.height

                Column {
                    id: form
                    anchors { top: parent.top; left: parent.left; right: parent.right }
                    spacing: pal.gap

                    Text {
                        width: parent.width
                        text: dlg.targetHost
                        color: pal.text
                        font.pixelSize: pal.fsBody
                        font.weight: Font.DemiBold
                        elide: Text.ElideRight
                    }

                    RLabel {
                        width: parent.width
                        section: true
                        text: root.tr("Rule type")
                    }
                    RSegment {
                        width: parent.width
                        options: dlg.types
                        currentIndex: dlg.typeIndex
                        onSelected: {
                            dlg.typeIndex = index
                            valField.text = dlg.suggest(index)
                        }
                    }

                    RLabel {
                        width: parent.width
                        text: root.tr("Value")
                    }
                    RField {
                        id: valField
                        width: parent.width
                        text: dlg.suggest(1)
                        placeholderText: root.tr("Value")
                        inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
                    }

                    RLabel {
                        width: parent.width
                        section: true
                        text: root.tr("Policy")
                    }
                    RSegment {
                        width: parent.width
                        options: dlg.policies
                        currentIndex: dlg.policyIndex
                        onSelected: dlg.policyIndex = index
                    }

                    RNote {
                        id: dlgMsg
                        width: parent.width
                        tone: "bad"
                    }

                    RButton {
                        width: parent.width
                        text: root.tr("Add")
                        variant: "primary"
                        onClicked: {
                            var payload = JSON.stringify({
                                type: dlg.types[dlg.typeIndex],
                                arg: valField.text.trim(),
                                policy: dlg.policies[dlg.policyIndex]
                            })
                            dlgMsg.text = root.tr("working…")
                            root.api("/rules/add", function(r, code) {
                                if (!r || r.error) { dlgMsg.text = (r && r.error) || ("HTTP " + code); return }
                                Popups.PopupUtils.close(dlg)
                                root.refresh()
                            }, "POST", payload)
                        }
                    }
                    RButton {
                        width: parent.width
                        text: root.tr("Cancel")
                        variant: "ghost"
                        onClicked: Popups.PopupUtils.close(dlg)
                    }
                }
            }
        }
    }
}
