import QtQuick 2.7
import Ubuntu.Components 1.3

Page {
    id: page
    property string confName: ""

    header: PageHeader {
        id: hdr
        title: root.tr("Edit config") + (page.confName ? ": " + page.confName : "")
        flickable: flick
        trailingActionBar.actions: [
            Action {
                iconName: "save"
                text: root.tr("Save")
                onTriggered: page.save()
            }
        ]
    }
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

    Flickable {
        id: flick
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        contentWidth: width
        contentHeight: col.height + units.gu(4)
        clip: true

        Column {
            id: col
            width: parent.width - units.gu(3)
            anchors { top: parent.top; topMargin: units.gu(1); horizontalCenter: parent.horizontalCenter }
            spacing: units.gu(1)

            Label { id: info; width: parent.width; fontSize: "small"; wrapMode: Text.Wrap }
            Label { id: msg; width: parent.width; fontSize: "small"; color: UbuntuColors.red; wrapMode: Text.Wrap }

            TextArea {
                id: area
                width: parent.width
                height: units.gu(40)
                inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                placeholderText: "[General]\nipv6 = false\n\n[Rule]\nFINAL,DIRECT"
            }

            Label {
                width: parent.width
                text: root.tr("unsupported lines")
                font.bold: true
                visible: skippedLabel.text !== ""
            }
            Label {
                id: skippedLabel
                width: parent.width
                wrapMode: Text.Wrap
                fontSize: "x-small"
                color: UbuntuColors.orange
            }
        }
    }
}
