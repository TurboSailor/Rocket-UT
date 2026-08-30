import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.ListItems 1.3 as ListItem
import Ubuntu.Components.Popups 1.3 as Popups

// Структурный редактор правил активного .conf: порядок в списке = приоритет.
Page {
    id: page

    property string confName: ""
    property var rules: []

    header: PageHeader {
        id: hdr
        title: root.tr("Rules")
        flickable: list
        trailingActionBar.actions: [
            Action {
                iconName: "add"
                text: root.tr("Add rule")
                onTriggered: Popups.PopupUtils.open(editDialog, page,
                    {mode: "add", ruleType: "DOMAIN-SUFFIX", ruleArg: "", rulePolicy: "PROXY"})
            },
            Action {
                iconName: "edit"
                text: root.tr("Edit as text")
                onTriggered: { confEdit.confName = page.confName; stack.push(confEdit) }
            }
        ]
    }

    onVisibleChanged: if (visible) reload()

    function reload() {
        root.api("/rules", function(r, code) {
            if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
            page.confName = r.conf || ""
            page.rules = r.rules || []
            msg.text = ""
            skipped.text = (r.skipped || []).join("\n")
        })
    }

    function absorbRules(r, code) {
        if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
        page.confName = r.conf || page.confName
        page.rules = r.rules || []
        skipped.text = (r.skipped || []).join("\n")
        msg.text = ""
        root.refresh()
    }

    function policyColor(p) {
        if (p === "PROXY") return UbuntuColors.blue
        if (p === "REJECT") return UbuntuColors.red
        return UbuntuColors.green
    }

    Label {
        anchors.centerIn: parent
        visible: page.rules.length === 0
        text: root.tr("no rules yet")
        color: UbuntuColors.graphite
    }

    ListView {
        id: list
        anchors { top: hdr.bottom; bottom: footer.top; left: parent.left; right: parent.right }
        model: page.rules
        clip: true
        delegate: ListItem.Empty {
            height: units.gu(8)
            showDivider: true

            Column {
                anchors {
                    left: parent.left; leftMargin: units.gu(2)
                    right: btns.left; rightMargin: units.gu(1)
                    verticalCenter: parent.verticalCenter
                }
                Label {
                    width: parent.width
                    elide: Text.ElideRight
                    text: (index + 1) + ".  " + modelData.type
                          + (modelData.type === "FINAL" ? "" : ",  " + modelData.arg)
                }
                Label {
                    fontSize: "small"
                    color: page.policyColor(modelData.policy)
                    text: "→ " + modelData.policy
                }
            }

            Row {
                id: btns
                anchors { right: parent.right; rightMargin: units.gu(1); verticalCenter: parent.verticalCenter }
                spacing: units.gu(0.5)
                Button {
                    text: "▲"
                    width: units.gu(3.5); height: units.gu(4)
                    enabled: index > 0
                    onClicked: root.api("/rules/move?line=" + modelData.line + "&dir=up",
                        page.absorbRules, "POST")
                }
                Button {
                    text: "▼"
                    width: units.gu(3.5); height: units.gu(4)
                    enabled: index < page.rules.length - 1
                    onClicked: root.api("/rules/move?line=" + modelData.line + "&dir=down",
                        page.absorbRules, "POST")
                }
                Button {
                    text: "✕"
                    width: units.gu(3.5); height: units.gu(4)
                    onClicked: root.api("/rules/delete?line=" + modelData.line,
                        page.absorbRules, "POST")
                }
            }

            onClicked: Popups.PopupUtils.open(editDialog, page, {
                mode: "update", line: modelData.line, ruleType: modelData.type,
                ruleArg: modelData.arg, rulePolicy: modelData.policy
            })
        }
    }

    Column {
        id: footer
        anchors { bottom: parent.bottom; left: parent.left; right: parent.right; margins: units.gu(1) }
        spacing: units.gu(0.5)
        Label {
            id: msg
            width: parent.width
            wrapMode: Text.Wrap
            fontSize: "small"
            color: UbuntuColors.red
        }
        Label {
            id: skipped
            width: parent.width
            wrapMode: Text.Wrap
            fontSize: "x-small"
            color: UbuntuColors.orange
        }
    }

    Component {
        id: editDialog
        Popups.Dialog {
            id: dlg
            property string mode: "add"
            property int line: 0
            property string ruleType: "DOMAIN-SUFFIX"
            property string ruleArg: ""
            property string rulePolicy: "PROXY"
            property var types: ["DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD",
                                 "IP-CIDR", "DST-PORT", "GEOIP", "FINAL"]
            property var policies: ["PROXY", "DIRECT", "REJECT"]

            title: dlg.mode === "add" ? root.tr("Add rule") : root.tr("Edit rule")

            OptionSelector {
                id: typeSel
                model: dlg.types
                selectedIndex: Math.max(0, dlg.types.indexOf(dlg.ruleType))
            }
            TextField {
                id: argField
                text: dlg.ruleArg
                visible: dlg.types[typeSel.selectedIndex] !== "FINAL"
                placeholderText: root.tr("Value")
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhPreferLowercase
            }
            OptionSelector {
                id: polSel
                model: dlg.policies
                selectedIndex: Math.max(0, dlg.policies.indexOf(dlg.rulePolicy))
            }
            Label {
                id: dlgMsg
                wrapMode: Text.Wrap
                fontSize: "small"
                color: UbuntuColors.red
            }
            Button {
                text: root.tr("Save")
                color: UbuntuColors.green
                onClicked: {
                    var payload = JSON.stringify({
                        type: dlg.types[typeSel.selectedIndex],
                        arg: argField.text.trim(),
                        policy: dlg.policies[polSel.selectedIndex]
                    })
                    var path = dlg.mode === "add"
                        ? "/rules/add" : "/rules/update?line=" + dlg.line
                    dlgMsg.text = root.tr("working…")
                    root.api(path, function(r, code) {
                        if (!r || r.error) {
                            dlgMsg.text = (r && r.error) || ("HTTP " + code)
                            return
                        }
                        page.absorbRules(r, code)
                        Popups.PopupUtils.close(dlg)
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
