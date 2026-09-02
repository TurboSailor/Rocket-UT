import QtQuick 2.7
import Ubuntu.Components 1.3

// Правка текста конфига: сводка разбора, ошибки, моноширинный редактор
// и список строк, которые демон не понял.
Page {
    id: page
    property string confName: ""

    onVisibleChanged: if (visible && page.confName !== "") load()

    function load() {
        msg.text = ""
        root.api("/conf?name=" + encodeURIComponent(page.confName), function(r, code) {
            if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
            area.text = r.text || ""
            info.text = r.rules + " " + root.tr("rules") + ", "
                      + r.proxies + " " + root.tr("proxies")
                      + (r.skipped && r.skipped.length
                         ? ",  " + r.skipped.length + " " + root.tr("unsupported lines") : "")
            skippedLabel.text = (r.skipped || []).join("\n")
        })
    }

    function save() {
        msg.text = root.tr("working…")
        root.api("/conf?name=" + encodeURIComponent(page.confName), function(r, code) {
            if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
            msg.text = ""
            info.text = r.rules + " " + root.tr("rules") + ", "
                      + r.proxies + " " + root.tr("proxies")
            skippedLabel.text = (r.skipped || []).join("\n")
            root.refresh()
        }, "POST", area.text)
    }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Edit config") + (page.confName ? ": " + page.confName : "")

        RIconButton {
            name: "check"
            tint: pal.ok
            onClicked: page.save()
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

            RNote {
                id: info
                width: parent.width
                tone: "info"
            }
            RNote {
                id: msg
                width: parent.width
                tone: "bad"
            }

            // --- редактор ---
            RCard {
                width: parent.width
                height: area.height + units.gu(2)

                TextArea {
                    id: area
                    anchors {
                        left: parent.left; leftMargin: units.gu(1)
                        right: parent.right; rightMargin: units.gu(1)
                        verticalCenter: parent.verticalCenter
                    }
                    height: units.gu(40)
                    color: pal.text
                    font.family: "Ubuntu Mono"
                    font.pixelSize: pal.fsSmall
                    inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                    placeholderText: "[General]\nipv6 = false\n\n[Rule]\nFINAL,DIRECT"
                }
            }

            // --- строки, которые демон не понял ---
            RLabel {
                width: parent.width
                section: true
                visible: skippedLabel.text !== ""
                text: root.tr("unsupported lines")
            }
            RNote {
                id: skippedLabel
                width: parent.width
                tone: "warn"
            }

            RButton {
                width: parent.width
                variant: "primary"
                text: root.tr("Save")
                onClicked: page.save()
            }
        }
    }
}
