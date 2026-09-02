import QtQuick 2.7
import Ubuntu.Components 1.3

// Подписки: карточка добавления источника и список подписок.
// Обновление одной подписки и удаление — явные кнопки в строке:
// свайп-действия UITK больше не используются.
Page {
    id: page

    onVisibleChanged: if (visible) root.api("/subs", function(r) { root.absorb(r) })

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Subscriptions")

        RIconButton {
            name: "refresh"
            tint: pal.text
            onClicked: {
                root.busy = root.tr("working…")
                root.api("/subs/update", function(r) {
                    root.busy = ""
                    root.absorb(r)
                    msg.text = (r && r.error) ? r.error : ""
                }, "POST")
            }
        }
    }

    // --- добавление подписки ---
    RCard {
        id: formCard
        anchors {
            top: hdr.bottom; topMargin: pal.pad
            left: parent.left; leftMargin: pal.pad
            right: parent.right; rightMargin: pal.pad
        }
        height: form.height + units.gu(3)

        Column {
            id: form
            anchors {
                left: parent.left; leftMargin: units.gu(1.5)
                right: parent.right; rightMargin: units.gu(1.5)
                verticalCenter: parent.verticalCenter
            }
            spacing: pal.gap

            RField {
                id: nameField
                width: parent.width
                placeholderText: root.tr("Name")
                inputMethodHints: Qt.ImhNoPredictiveText
            }
            RField {
                id: urlField
                width: parent.width
                placeholderText: root.tr("URL")
                inputMethodHints: Qt.ImhUrlCharactersOnly | Qt.ImhNoPredictiveText
            }
            RButton {
                width: parent.width
                variant: "primary"
                text: root.tr("Add")
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
            RNote {
                id: msg
                width: parent.width
                tone: "warn"
            }
        }
    }

    // --- список подписок ---
    ListView {
        id: list
        anchors {
            top: formCard.bottom
            bottom: parent.bottom
            left: parent.left
            right: parent.right
        }
        model: root.subs
        topMargin: pal.gap
        bottomMargin: pal.pad
        spacing: units.gu(1)
        clip: true

        delegate: RRow {
            x: pal.pad
            width: list.width - 2 * pal.pad
            icon: "link"
            tint: pal.violet
            title: modelData.name
            subtitle: modelData.count + " " + root.tr("nodes")
                      + (modelData.err ? "  ·  " + modelData.err : "")
            value: root.fmtAgo(modelData.updated_at)
            valueColor: modelData.err ? pal.bad : pal.dim

            RIconButton {
                name: "refresh"
                tint: pal.accent
                onClicked: {
                    root.busy = root.tr("working…")
                    root.api("/subs/update?id=" + modelData.id, function(r) {
                        root.busy = ""
                        root.absorb(r)
                        root.refresh()
                    }, "POST")
                }
            }
            RIconButton {
                name: "trash"
                tint: pal.bad
                onClicked: root.api("/subs/delete?id=" + modelData.id, function(r) {
                    root.absorb(r)
                    root.refresh()
                }, "POST")
            }
        }
    }

    REmpty {
        anchors.centerIn: list
        visible: root.subs.length === 0
        icon: "link"
        text: root.tr("no subscriptions yet")
    }
}
