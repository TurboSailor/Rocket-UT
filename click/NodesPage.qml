import QtQuick 2.7
import Ubuntu.Components 1.3

// Список узлов. Тап — активировать; свайп влево — правка и удаление
// (ListItem из Ubuntu.Components 1.3, а не ListItems.Subtitled: у последнего
// нет ни actions, ни свойства control).
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

    function activate(id) {
        root.busy = root.tr("working…")
        root.api("/nodes/select?id=" + id, function(r) {
            root.busy = ""
            root.absorb(r)
        }, "POST")
    }

    function edit(n) {
        nodeEdit.node = n
        stack.push(nodeEdit)
    }

    function remove(id) {
        root.api("/nodes/delete?id=" + id, function(r) {
            root.absorb(r)
            root.refresh()
        }, "POST")
    }

    function latencyText(n) {
        if (n.latency > 0) return n.latency + " ms"
        if (n.latency === 0) return ""
        return "—"
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

        delegate: ListItem {
            id: row
            height: units.gu(8)

            onClicked: page.activate(modelData.id)

            leadingActions: ListItemActions {
                actions: [
                    Action {
                        iconName: "delete"
                        text: root.tr("Delete")
                        onTriggered: page.remove(modelData.id)
                    }
                ]
            }
            trailingActions: ListItemActions {
                actions: [
                    Action {
                        iconName: "edit"
                        text: root.tr("Edit")
                        onTriggered: page.edit(modelData)
                    },
                    Action {
                        iconName: "info"
                        text: root.tr("Test")
                        onTriggered: root.api("/nodes/test?id=" + modelData.id, function(r) {
                            if (r && r.nodes) root.nodes = r.nodes
                        }, "POST")
                    }
                ]
            }

            ListItemLayout {
                title.text: modelData.name
                        + (modelData.id === root.st.node_id ? "   ●" : "")
                subtitle.text: modelData.type + "  " + modelData.server
                        + (modelData.port ? ":" + modelData.port : "")
                summary.text: page.latencyText(modelData)
                        + (modelData.sub_id ? "   " + root.tr("from subscription") : "")

                // Кнопка правки видна без свайпа: свайп находят не все.
                Icon {
                    SlotsLayout.position: SlotsLayout.Trailing
                    width: units.gu(2.5)
                    height: width
                    name: "edit"
                    MouseArea {
                        anchors.fill: parent
                        anchors.margins: units.gu(-1)
                        onClicked: page.edit(modelData)
                    }
                }
            }
        }
    }
}
