import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.Popups 1.3 as Popups

// Структурный редактор правил активного .conf: порядок в списке = приоритет.
Page {
    id: page

    property string confName: ""
    property var rules: []

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
        title: root.tr("Rules")

        RIconButton {
            name: "plus"
            tint: pal.accent
            onClicked: Popups.PopupUtils.open(editDialog, page,
                {mode: "add", ruleType: "DOMAIN-SUFFIX", ruleArg: "", rulePolicy: "PROXY"})
        }
        RIconButton {
            name: "edit"
            tint: pal.text
            onClicked: { confEdit.confName = page.confName; stack.push(confEdit) }
        }
    }

    REmpty {
        anchors.centerIn: parent
        visible: page.rules.length === 0
        icon: "filter"
        text: root.tr("no rules yet")
    }

    ListView {
        id: list
        anchors { top: hdr.bottom; bottom: footer.top; left: parent.left; right: parent.right }
        model: page.rules
        topMargin: pal.pad
        bottomMargin: pal.pad
        spacing: units.gu(1)
        clip: true

        delegate: RRow {
            x: pal.pad
            width: list.width - 2 * pal.pad
            icon: page.policyIcon(modelData.policy)
            tint: page.policyColor(modelData.policy)
            title: (index + 1) + ". " + modelData.type
                   + (modelData.type === "FINAL" ? "" : ", " + modelData.arg)
            subtitle: "→ " + modelData.policy

            RIconButton {
                name: "chevronUp"
                tint: pal.dim
                iconSize: units.gu(2.2)
                enabled: index > 0
                onClicked: root.api("/rules/move?line=" + modelData.line + "&dir=up",
                    page.absorbRules, "POST")
            }
            RIconButton {
                name: "chevronDown"
                tint: pal.dim
                iconSize: units.gu(2.2)
                enabled: index < page.rules.length - 1
                onClicked: root.api("/rules/move?line=" + modelData.line + "&dir=down",
                    page.absorbRules, "POST")
            }
            RIconButton {
                name: "trash"
                tint: pal.bad
                iconSize: units.gu(2.2)
                onClicked: root.api("/rules/delete?line=" + modelData.line,
                    page.absorbRules, "POST")
            }

            onClicked: Popups.PopupUtils.open(editDialog, page, {
                mode: "update", line: modelData.line, ruleType: modelData.type,
                ruleArg: modelData.arg, rulePolicy: modelData.policy
            })
        }
    }

    Column {
        id: footer
        anchors {
            bottom: parent.bottom
            left: parent.left
            right: parent.right
            margins: pal.pad
        }
        spacing: units.gu(1)

        RNote {
            id: msg
            width: parent.width
            tone: "bad"
        }
        RNote {
            id: skipped
            width: parent.width
            tone: "warn"
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

            Item {
                id: body
                width: parent ? parent.width : units.gu(34)
                height: form.height

                Column {
                    id: form
                    anchors { top: parent.top; left: parent.left; right: parent.right }
                    spacing: pal.gap

                    RLabel {
                        width: parent.width
                        section: true
                        text: root.tr("Rule type")
                    }

                    // Семь типов в сегменты не влезают: сетка кнопок 2×N.
                    Grid {
                        width: parent.width
                        columns: 2
                        spacing: pal.gap

                        Repeater {
                            model: dlg.types

                            delegate: RButton {
                                width: (form.width - pal.gap) / 2
                                text: modelData
                                variant: modelData === dlg.ruleType ? "primary" : "ghost"
                                onClicked: dlg.ruleType = modelData
                            }
                        }
                    }

                    RLabel {
                        width: parent.width
                        visible: dlg.ruleType !== "FINAL"
                        text: root.tr("Value")
                    }
                    RField {
                        id: argField
                        width: parent.width
                        visible: dlg.ruleType !== "FINAL"
                        text: dlg.ruleArg
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
                        currentIndex: Math.max(0, dlg.policies.indexOf(dlg.rulePolicy))
                        onSelected: dlg.rulePolicy = dlg.policies[index]
                    }

                    RNote {
                        id: dlgMsg
                        width: parent.width
                        tone: "bad"
                    }

                    RButton {
                        width: parent.width
                        text: root.tr("Save")
                        variant: "primary"
                        onClicked: {
                            var payload = JSON.stringify({
                                type: dlg.ruleType,
                                arg: argField.text.trim(),
                                policy: dlg.rulePolicy
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
