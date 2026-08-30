import QtQuick 2.7
import Ubuntu.Components 1.3
import Ubuntu.Components.ListItems 1.3 as ListItem

Page {
    id: page

    header: PageHeader {
        id: hdr
        title: dirModel.cur
        flickable: fileList
        trailingActionBar.actions: [
            Action {
                iconName: "up"
                text: root.tr("Up")
                enabled: dirModel.cur !== "/home/phablet" && dirModel.cur !== "/"
                onTriggered: dirModel.up()
            }
        ]
    }
    onVisibleChanged: if (visible) dirModel.load()

    ListModel {
        id: dirModel
        property string cur: root.startDir

        function up() {
            if (cur === "/home/phablet" || cur === "/") return
            cur = cur.substring(0, cur.lastIndexOf("/")) || "/"
            load()
        }

        function load() {
            clear()
            root.api("/listdir?path=" + encodeURIComponent(cur), function(r) {
                clear()
                if (!r || r.error) return
                var dirs = r.dirs || [], files = r.files || [], i
                for (i = 0; i < dirs.length; i++)
                    append({name: dirs[i], isDir: true, path: root.joinPath(cur, dirs[i])})
                for (i = 0; i < files.length; i++)
                    append({name: files[i], isDir: false, path: root.joinPath(cur, files[i])})
            })
        }
    }

    ListView {
        id: fileList
        anchors { top: hdr.bottom; bottom: parent.bottom; left: parent.left; right: parent.right }
        model: dirModel
        clip: true
        delegate: ListItem.Standard {
            text: (model.isDir ? "📁  " : "📄  ") + model.name
            onClicked: {
                if (model.isDir) {
                    dirModel.cur = model.path
                    dirModel.load()
                    return
                }
                // Картинка распознаётся демоном через zxing, а не читается как текст.
                if (root.pendingFile === "qr") {
                    msg.text = root.tr("working…")
                    root.api("/qrdecode", function(r, code) {
                        if (!r || !r.text) {
                            msg.text = (r && r.error) || ("HTTP " + code)
                            return
                        }
                        msg.text = ""
                        importPage.setText(r.text)
                        stack.pop()
                    }, "POST", model.path)
                    return
                }
                root.api("/readfile", function(r, code) {
                    if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
                    if (root.pendingFile === "conf") {
                        // Файл .conf сохраняем под именем файла без расширения.
                        var base = model.name.replace(/\.[^.]+$/, "")
                        root.api("/conf?name=" + encodeURIComponent(base), function(r2, c2) {
                            if (!r2 || r2.error) { msg.text = (r2 && r2.error) || ("HTTP " + c2); return }
                            stack.pop()
                            confsPage.reload()
                            root.refresh()
                        }, "POST", r.text)
                    } else {
                        importPage.setText(r.text)
                        stack.pop()
                    }
                }, "POST", model.path)
            }
        }
    }

    Label {
        id: msg
        anchors { bottom: parent.bottom; left: parent.left; right: parent.right; margins: units.gu(2) }
        wrapMode: Text.Wrap
        fontSize: "small"
        color: UbuntuColors.red
    }
}
