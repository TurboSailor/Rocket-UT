import QtQuick 2.7
import Ubuntu.Components 1.3

// Главный экран: круглая кнопка питания, состояние туннеля, режим
// маршрутизации и плитки разделов.
Page {
    id: page

    // tone — цвет состояния, от него красится вся геройская карточка.
    readonly property color tone: root.st.error ? pal.warn
                                : (root.st.up ? pal.ok : pal.faint)

    function toggle() {
        if (root.busy !== "") return
        root.busy = root.tr("working…")
        root.api(root.st.up ? "/down" : "/up", function(r) {
            root.busy = ""
            root.absorb(r)
        }, "POST")
    }

    function setMode(v) {
        if (v === root.st.mode) return
        root.busy = root.tr("working…")
        root.api("/mode?v=" + v, function(r) { root.busy = ""; root.absorb(r) }, "POST")
    }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Rocket")
        back: false
        icon: "bolt"
    }

    Flickable {
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

            // --- состояние туннеля ---
            RCard {
                id: hero
                width: parent.width
                height: units.gu(33)

                Item {
                    id: power
                    width: Math.min(units.gu(17), hero.width * 0.45)
                    height: width
                    anchors {
                        horizontalCenter: parent.horizontalCenter
                        top: parent.top
                        topMargin: units.gu(2.5)
                    }

                    Rectangle {
                        anchors.fill: parent
                        radius: width / 2
                        color: Qt.rgba(page.tone.r, page.tone.g, page.tone.b, 0.07)
                        border.width: 1
                        border.color: Qt.rgba(page.tone.r, page.tone.g, page.tone.b, 0.3)
                    }

                    Rectangle {
                        id: disc
                        anchors.centerIn: parent
                        width: parent.width - units.gu(3)
                        height: width
                        radius: width / 2
                        color: Qt.rgba(page.tone.r, page.tone.g, page.tone.b, root.st.up ? 0.2 : 0.1)
                        border.width: units.dp(1.5)
                        border.color: page.tone
                        opacity: touch.pressed ? 0.65 : 1

                        RIcon {
                            anchors.centerIn: parent
                            name: "power"
                            tint: page.tone
                            size: parent.width * 0.45
                            weight: units.dp(1.3)
                        }
                    }

                    MouseArea {
                        id: touch
                        anchors.fill: parent
                        enabled: root.busy === ""
                        onClicked: page.toggle()
                    }
                }

                Text {
                    id: stateText
                    anchors {
                        left: parent.left; leftMargin: units.gu(1.5)
                        right: parent.right; rightMargin: units.gu(1.5)
                        top: power.bottom; topMargin: units.gu(1.5)
                    }
                    horizontalAlignment: Text.AlignHCenter
                    text: root.busy !== "" ? root.busy
                         : (root.st.up ? root.tr("CONNECTED") : root.tr("DISCONNECTED"))
                    color: page.tone
                    font.pixelSize: pal.fsTitle
                    font.weight: Font.DemiBold
                    font.letterSpacing: units.dp(0.5)
                    elide: Text.ElideRight
                }

                Text {
                    anchors {
                        left: parent.left; leftMargin: units.gu(2)
                        right: parent.right; rightMargin: units.gu(2)
                        top: stateText.bottom; topMargin: units.gu(0.5)
                    }
                    horizontalAlignment: Text.AlignHCenter
                    wrapMode: Text.Wrap
                    maximumLineCount: 2
                    elide: Text.ElideRight
                    text: root.st.error ? root.st.error
                         : (root.st.node ? root.nodeLabel(root.st.node)
                                         : root.tr("no node selected"))
                    color: pal.dim
                    font.pixelSize: pal.fsSmall
                }

                // --- трафик ---
                Rectangle {
                    id: sep
                    anchors {
                        left: parent.left; leftMargin: units.gu(1.5)
                        right: parent.right; rightMargin: units.gu(1.5)
                        bottom: traffic.top; bottomMargin: units.gu(1)
                    }
                    height: 1
                    color: pal.border
                }

                Row {
                    id: traffic
                    anchors {
                        left: parent.left; leftMargin: units.gu(1.5)
                        right: parent.right; rightMargin: units.gu(1.5)
                        bottom: parent.bottom; bottomMargin: units.gu(1.5)
                    }
                    height: units.gu(3)

                    Row {
                        width: (traffic.width - units.gu(1)) / 2
                        height: parent.height
                        spacing: units.gu(0.75)

                        RIcon {
                            anchors.verticalCenter: parent.verticalCenter
                            name: "download"
                            tint: pal.accent
                            size: units.gu(2)
                        }
                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            text: root.fmtBytes(root.st.rx)
                            color: pal.text
                            font.pixelSize: pal.fsSmall
                        }
                    }
                    Item { width: units.gu(1); height: 1 }
                    Row {
                        width: (traffic.width - units.gu(1)) / 2
                        height: parent.height
                        spacing: units.gu(0.75)
                        layoutDirection: Qt.RightToLeft

                        Text {
                            anchors.verticalCenter: parent.verticalCenter
                            text: root.fmtBytes(root.st.tx)
                            color: pal.text
                            font.pixelSize: pal.fsSmall
                        }
                        RIcon {
                            anchors.verticalCenter: parent.verticalCenter
                            name: "upload"
                            tint: pal.violet
                            size: units.gu(2)
                        }
                    }
                }
            }

            // --- режим маршрутизации ---
            RLabel {
                width: parent.width
                section: true
                text: root.tr("Routing")
            }

            RSegment {
                width: parent.width
                options: [root.tr("Rules"), root.tr("Proxy"), root.tr("Direct")]
                currentIndex: root.st.mode === "proxy" ? 1 : (root.st.mode === "direct" ? 2 : 0)
                onSelected: page.setMode(index === 1 ? "proxy" : (index === 2 ? "direct" : "config"))
            }

            // --- разделы ---
            RLabel {
                width: parent.width
                section: true
                text: root.tr("Sections")
            }

            Grid {
                width: parent.width
                columns: 2
                spacing: pal.gap

                RTile {
                    width: (parent.width - pal.gap) / 2
                    icon: "server"
                    tint: pal.accent
                    label: root.tr("Nodes")
                    value: String(root.nodes.length)
                    onClicked: stack.push(nodesPage)
                }
                RTile {
                    width: (parent.width - pal.gap) / 2
                    icon: "link"
                    tint: pal.violet
                    label: root.tr("Subscriptions")
                    value: String(root.subs.length)
                    onClicked: stack.push(subsPage)
                }
                RTile {
                    width: (parent.width - pal.gap) / 2
                    icon: "download"
                    tint: pal.ok
                    label: root.tr("Import")
                    onClicked: stack.push(importPage)
                }
                RTile {
                    width: (parent.width - pal.gap) / 2
                    icon: "filter"
                    tint: pal.warn
                    label: root.tr("Rules")
                    onClicked: stack.push(rulesPage)
                }
                RTile {
                    width: (parent.width - pal.gap) / 2
                    icon: "file"
                    tint: pal.dim
                    label: root.tr("Configs")
                    onClicked: stack.push(confsPage)
                }
                RTile {
                    width: (parent.width - pal.gap) / 2
                    icon: "clock"
                    tint: pal.accent
                    label: root.tr("Log")
                    onClicked: stack.push(logPage)
                }
            }

            // --- предупреждения демона ---
            RNote {
                width: parent.width
                tone: "info"
                text: root.st.handshake_age !== undefined
                      ? "awg0 handshake " + root.st.handshake_age + "s" : ""
            }
            RNote {
                width: parent.width
                tone: "warn"
                text: (root.st.skipped !== undefined && root.st.skipped.length > 0)
                      ? root.st.skipped.length + " " + root.tr("unsupported lines") : ""
            }
        }
    }
}
