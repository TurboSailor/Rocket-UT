import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.ListItems 1.3 as ListItem

Page {
    id: page

    header: PageHeader {
        id: hdr
        title: root.tr("Nodes")
        flickable: list
        trailingActionBar.actions: [
            Action {
                iconName: "add"
                text: root.tr("Import")
                onTriggered: stack.push(importPage)
            },
            Action {
                iconName: "reload"
                text: root.tr("Test all")
                onTriggered: {
                    root.busy = root.tr("testing…")
                    root.api("/nodes/test", function(r) {
                        root.busy = ""
                        if (r && r.nodes) root.nodes = r.nodes
                    }, "POST")
                }
            }
        ]
    }

    Label {
        anchors.centerIn: parent
        visible: root.nodes.length === 0
        text: root.tr("no nodes yet")
        color: UbuntuColors.graphite
    }

    ListView {
        id: list
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        model: root.nodes
        clip: true

        delegate: ListItem.Subtitled {
            text: modelData.name + (modelData.id === root.st.node_id ? "  ●" : "")
            subText: modelData.type + "  " + modelData.server + ":" + modelData.port
                     + (modelData.latency > 0 ? "   " + modelData.latency + " ms"
                        : (modelData.latency === 0 ? "" : "   —"))
            selected: modelData.id === root.st.node_id
            onClicked: {
                root.busy = root.tr("working…")
                root.api("/nodes/select?id=" + modelData.id, function(r) {
                    root.busy = ""
                    root.absorb(r)
                }, "POST")
            }
            // У ListItem.Subtitled нет свойства control — удаление свайпом.
            removable: true
            confirmRemoval: true
            onItemRemoved: root.api("/nodes/delete?id=" + modelData.id, function(r) {
                root.absorb(r)
                root.refresh()
            }, "POST")
        }
    }
}
