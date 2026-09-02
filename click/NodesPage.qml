import QtQuick 2.7
import Ubuntu.Components 1.3

// Список узлов. Тап по строке активирует узел, правка и удаление вынесены
// в явные кнопки строки: свайп-действия UITK находят не все.
Page {
    id: page

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

    // Цвет латентности: быстрый узел виден на глаз, мёртвый — приглушён.
    function latencyColor(n) {
        if (n.latency > 0) {
            if (n.latency < 150) return pal.ok
            if (n.latency < 400) return pal.warn
            return pal.dim
        }
        if (n.latency === 0) return pal.dim
        return pal.faint
    }

    function nodeIcon(t) {
        if (t === "ssh") return "terminal"
        if (t === "awg") return "shield"
        if (t === "socks5" || t === "http") return "globe"
        return "server"
    }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Nodes")

        RIconButton {
            name: "plus"
            tint: pal.text
            onClicked: stack.push(importPage)
        }
        RIconButton {
            name: "refresh"
            tint: pal.text
            onClicked: {
                root.busy = root.tr("testing…")
                root.api("/nodes/test", function(r) {
                    root.busy = ""
                    if (r && r.nodes) root.nodes = r.nodes
                }, "POST")
            }
        }
    }

    REmpty {
        anchors.centerIn: parent
        visible: root.nodes.length === 0
        icon: "server"
        text: root.tr("no nodes yet")
    }

    ListView {
        id: list
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        model: root.nodes
        topMargin: pal.pad
        bottomMargin: pal.pad
        spacing: units.gu(1)
        clip: true

        delegate: RRow {
            x: pal.pad
            width: list.width - 2 * pal.pad

            icon: page.nodeIcon(modelData.type)
            tint: root.typeColor(modelData.type)
            title: modelData.name
            subtitle: modelData.type + "  " + modelData.server
                    + (modelData.port ? ":" + modelData.port : "")
                    + (modelData.sub_id ? "  ·  " + root.tr("from subscription") : "")
            value: page.latencyText(modelData)
            valueColor: page.latencyColor(modelData)
            active: modelData.id === root.st.node_id

            onClicked: page.activate(modelData.id)

            RIconButton {
                name: "edit"
                tint: pal.dim
                onClicked: page.edit(modelData)
            }
            RIconButton {
                name: "trash"
                tint: pal.bad
                onClicked: page.remove(modelData.id)
            }
        }
    }
}
