import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.ListItems 1.3 as ListItem

Page {
    id: page

    header: PageHeader {
        id: hdr
        title: root.tr("Subscriptions")
        trailingActionBar.actions: [
            Action {
                iconName: "reload"
                text: root.tr("Refresh")
                onTriggered: {
                    root.busy = root.tr("working…")
                    root.api("/subs/update", function(r) {
                        root.busy = ""
                        root.absorb(r)
                        msg.text = (r && r.error) ? r.error : ""
                    }, "POST")
                }
            }
        ]
    }
    onVisibleChanged: if (visible) root.api("/subs", function(r) { root.absorb(r) })

    Column {
        id: form
        anchors { top: hdr.bottom; left: parent.left; right: parent.right; margins: units.gu(2) }
        spacing: units.gu(1)

        TextField {
            id: nameField
            width: parent.width
            placeholderText: root.tr("Name")
            inputMethodHints: Qt.ImhNoPredictiveText
        }
        TextField {
            id: urlField
            width: parent.width
            placeholderText: root.tr("URL")
            inputMethodHints: Qt.ImhUrlCharactersOnly | Qt.ImhNoPredictiveText
        }
        Button {
            width: parent.width
            text: root.tr("Add")
            color: UbuntuColors.green
            enabled: urlField.text.trim() !== ""
            onClicked: {
                msg.text = root.tr("working…")
                root.api("/subs/add?name=" + encodeURIComponent(nameField.text.trim() || "sub")
                         + "&url=" + encodeURIComponent(urlField.text.trim()),
                    function(r, code) {
                        root.absorb(r)
                        if (r && r.error) { msg.text = r.error; return }
                        msg.text = r ? (r.count + " " + root.tr("nodes")) : ("HTTP " + code)
                        urlField.text = ""
                        root.refresh()
                    }, "POST")
            }
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
        visible: root.subs.length === 0
        text: root.tr("no subscriptions yet")
        color: UbuntuColors.graphite
    }

    ListView {
        anchors { top: form.bottom; topMargin: units.gu(1); bottom: parent.bottom
                  left: parent.left; right: parent.right }
        model: root.subs
        clip: true
        delegate: ListItem.Subtitled {
            text: modelData.name
            subText: modelData.count + " " + root.tr("nodes")
                     + "   " + root.fmtAgo(modelData.updated_at)
                     + (modelData.err ? "   " + modelData.err : "")
            onClicked: {
                root.busy = root.tr("working…")
                root.api("/subs/update?id=" + modelData.id, function(r) {
                    root.busy = ""
                    root.absorb(r)
                    root.refresh()
                }, "POST")
            }
            // У ListItem.Subtitled нет свойства control — удаление свайпом.
            removable: true
            confirmRemoval: true
            onItemRemoved: root.api("/subs/delete?id=" + modelData.id, function(r) {
                root.absorb(r)
                root.refresh()
            }, "POST")
        }
    }
}
