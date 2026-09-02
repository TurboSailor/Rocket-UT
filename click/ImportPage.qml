import QtQuick 2.7
import Ubuntu.Components 1.3

// Импорт: три источника (сканер, картинка с QR, файл) плюс ручная вставка
// текста конфига/ссылок в многострочное поле.
Page {
    id: page

    // setText вызывается из FilePage после чтения файла.
    function setText(t) { area.text = t }

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Import")
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

            RNote {
                width: parent.width
                tone: "info"
                text: root.tr("Import from file or paste text below")
            }

            // Список форматов длинный: обычный Text с переносом, потому что
            // RLabel обрезает по ширине.
            Text {
                width: parent.width
                wrapMode: Text.Wrap
                color: pal.faint
                font.pixelSize: pal.fsTiny
                text: root.tr("Supported: vless, vmess, ss, trojan, socks5, http, ssh URIs, sub:// links, Shadowrocket .conf, AmneziaWG .conf")
            }

            // --- источники ---
            RButton {
                width: parent.width
                icon: "camera"
                text: root.tr("Scan QR")
                variant: "primary"
                onClicked: stack.push(scanPage)
            }
            RButton {
                width: parent.width
                icon: "file"
                text: root.tr("QR from image")
                variant: "ghost"
                onClicked: { root.pendingFile = "qr"; stack.push(filePage) }
            }
            RButton {
                width: parent.width
                icon: "folder"
                text: root.tr("Open file")
                variant: "ghost"
                onClicked: { root.pendingFile = "import"; stack.push(filePage) }
            }

            RLabel {
                width: parent.width
                text: root.tr("AmneziaWG name")
            }
            RField {
                id: nameField
                width: parent.width
                text: "awg"
                placeholderText: root.tr("AmneziaWG name")
                maximumLength: 32
                inputMethodHints: Qt.ImhPreferLowercase | Qt.ImhNoPredictiveText
            }

            // Многострочный ввод — TextArea из UITK (SuruDark), в тёмной
            // карточке-подложке, чтобы рамка совпадала с остальной формой.
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
                    height: units.gu(26)
                    font.family: "Ubuntu Mono"
                    font.pixelSize: pal.fsSmall
                    inputMethodHints: Qt.ImhNoPredictiveText | Qt.ImhNoAutoUppercase
                    placeholderText: root.tr("Paste links or config")
                }
            }

            RNote {
                id: msg
                width: parent.width
                tone: "bad"
            }

            RButton {
                width: parent.width
                icon: "download"
                text: root.tr("Import")
                variant: "success"
                enabled: area.text.trim() !== ""
                onClicked: {
                    msg.text = root.tr("working…")
                    root.api("/nodes/import?name=" + encodeURIComponent(nameField.text.trim() || "awg"),
                        function(r, code) {
                            if (!r || r.error) {
                                msg.text = (r && r.error) || ("HTTP " + code)
                                return
                            }
                            msg.text = ""
                            root.absorb(r)
                            root.refresh()
                            area.text = ""
                            stack.pop()
                        }, "POST", area.text)
                }
            }
        }
    }
}
