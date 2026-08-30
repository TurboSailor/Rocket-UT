import QtQuick 2.7
import Ubuntu.Components 1.3

Page {
    id: page

    header: PageHeader {
        id: hdr
        // Иконка приложения в заголовке: title рисует только текст,
        // поэтому заголовок собираем сами через contents.
        contents: Row {
            anchors.verticalCenter: parent.verticalCenter
            spacing: units.gu(1)
            Image {
                source: Qt.resolvedUrl("icon.png")
                width: units.gu(3.5)
                height: width
                sourceSize.width: units.gu(3.5) * 2
                sourceSize.height: units.gu(3.5) * 2
                fillMode: Image.PreserveAspectFit
                smooth: true
                anchors.verticalCenter: parent.verticalCenter
            }
            Label {
                text: root.tr("Rocket")
                fontSize: "large"
                anchors.verticalCenter: parent.verticalCenter
            }
        }
        trailingActionBar.actions: [
            Action {
                iconName: "filters"
                text: root.tr("Rules")
                onTriggered: stack.push(rulesPage)
            },
            Action {
                iconName: "settings"
                text: root.tr("Configs")
                onTriggered: stack.push(confsPage)
            },
            Action {
                iconName: "history"
                text: root.tr("Log")
                onTriggered: stack.push(logPage)
            }
        ]
    }

    Flickable {
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        contentWidth: width
        contentHeight: col.height + units.gu(4)
        clip: true

        Column {
            id: col
            width: parent.width - units.gu(4)
            anchors { top: parent.top; topMargin: units.gu(2); horizontalCenter: parent.horizontalCenter }
            spacing: units.gu(1.5)

            UbuntuShape {
                width: parent.width
                height: units.gu(13)
                backgroundColor: root.st.error ? UbuntuColors.graphite
                                : (root.st.up ? UbuntuColors.green : UbuntuColors.red)
                Column {
                    anchors.centerIn: parent
                    spacing: units.gu(0.5)
                    Label {
                        anchors.horizontalCenter: parent.horizontalCenter
                        text: root.st.up ? root.tr("CONNECTED") : root.tr("DISCONNECTED")
                        color: "white"; fontSize: "x-large"; font.bold: true
                    }
                    Label {
                        anchors.horizontalCenter: parent.horizontalCenter
                        width: col.width - units.gu(2)
                        horizontalAlignment: Text.AlignHCenter
                        wrapMode: Text.Wrap
                        text: root.st.error ? root.st.error
                             : (root.st.node ? root.nodeLabel(root.st.node) : root.tr("no node selected"))
                        color: "white"; fontSize: "small"; opacity: 0.9
                    }
                }
            }

            Switch {
                anchors.horizontalCenter: parent.horizontalCenter
                checked: root.st.up === true
                enabled: root.busy === ""
                onClicked: {
                    root.busy = root.tr("working…")
                    root.api(checked ? "/up" : "/down", function(r) {
                        root.busy = ""
                        root.absorb(r)
                    }, "POST")
                }
            }

            Label {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                text: root.busy !== "" ? root.busy
                      : (root.st.up ? "↓ " + root.fmtBytes(root.st.rx) + "   ↑ " + root.fmtBytes(root.st.tx) : " ")
            }

            Label {
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                fontSize: "small"
                color: UbuntuColors.graphite
                visible: root.st.handshake_age !== undefined
                text: root.st.handshake_age !== undefined
                      ? "awg0 handshake: " + root.st.handshake_age + "s" : ""
            }

            Label { text: root.tr("Routing"); font.bold: true }

            OptionSelector {
                id: modeSel
                width: parent.width
                model: [root.tr("Config rules"), root.tr("Proxy all"), root.tr("Direct")]
                // Индекс синхронизируется со статусом, но не перебивает выбор пользователя.
                selectedIndex: root.st.mode === "proxy" ? 1 : (root.st.mode === "direct" ? 2 : 0)
                onDelegateClicked: {
                    var v = index === 1 ? "proxy" : (index === 2 ? "direct" : "config")
                    if (v === root.st.mode) return
                    root.busy = root.tr("working…")
                    root.api("/mode?v=" + v, function(r) { root.busy = ""; root.absorb(r) }, "POST")
                }
            }

            Button {
                width: parent.width
                text: root.tr("Nodes") + "  (" + root.nodes.length + ")"
                color: UbuntuColors.blue
                onClicked: stack.push(nodesPage)
            }
            Button {
                width: parent.width
                text: root.tr("Subscriptions")
                onClicked: stack.push(subsPage)
            }
            Button {
                width: parent.width
                text: root.tr("Import")
                color: UbuntuColors.green
                onClicked: stack.push(importPage)
            }

            Label {
                width: parent.width
                wrapMode: Text.Wrap
                fontSize: "small"
                color: UbuntuColors.orange
                visible: root.st.skipped !== undefined && root.st.skipped.length > 0
                text: visible ? (root.st.skipped.length + " " + root.tr("unsupported lines")) : ""
            }
        }
    }
}
