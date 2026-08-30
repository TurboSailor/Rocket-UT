import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.ListItems 1.3 as ListItem
import Ubuntu.Components.Popups 1.3 as Popups

Page {
    id: page

    header: PageHeader {
        id: hdr
        title: root.tr("Log")
        flickable: list
    }

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

    Label {
        anchors.centerIn: parent
        visible: entries.count === 0
        text: root.tr("no traffic yet")
        color: UbuntuColors.graphite
    }

    ListView {
        id: list
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        model: entries
        clip: true
        delegate: ListItem.Subtitled {
            text: (model.host || model.ip) + (model.open ? "" : "  ·")
            subText: root.fmtTime(model.time) + "   " + model.net + ":" + model.port
                     + "   → " + model.policy
                     + "   ↓" + root.fmtBytes(model.down) + " ↑" + root.fmtBytes(model.up)
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
            title: root.tr("Add rule")
            text: targetHost

            OptionSelector {
                id: typeSel
                model: ["DOMAIN", "DOMAIN-SUFFIX", "IP-CIDR"]
                selectedIndex: 1
                onDelegateClicked: valField.text = dlg.suggest(index)
            }

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

            TextField {
                id: valField
                text: dlg.suggest(1)
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }

            OptionSelector {
                id: polSel
                model: ["PROXY", "DIRECT", "REJECT"]
                selectedIndex: 0
            }

            Label {
                id: dlgMsg
                wrapMode: Text.Wrap
                fontSize: "small"
                color: UbuntuColors.red
            }

            Button {
                text: root.tr("Add")
                color: UbuntuColors.green
                onClicked: {
                    var payload = JSON.stringify({
                        type: typeSel.model[typeSel.selectedIndex],
                        arg: valField.text.trim(),
                        policy: polSel.model[polSel.selectedIndex]
                    })
                    dlgMsg.text = root.tr("working…")
                    root.api("/rules/add", function(r, code) {
                        if (!r || r.error) { dlgMsg.text = (r && r.error) || ("HTTP " + code); return }
                        Popups.PopupUtils.close(dlg)
                        root.refresh()
                    }, "POST", payload)
                }
            }
            Button {
                text: root.tr("Cancel")
                onClicked: Popups.PopupUtils.close(dlg)
            }
        }
    }
}
