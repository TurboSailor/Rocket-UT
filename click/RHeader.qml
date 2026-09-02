import QtQuick 2.7
import Ubuntu.Components 1.3

// Шапка страницы: кнопка «назад», заголовок, справа — переданные кнопки.
// Заголовок ограничен левой гранью блока кнопок (elide), поэтому не уезжает
// за экран ни на 32, ни на 60 gu.
Item {
    id: hdr

    property string title: ""
    property bool back: true
    property string icon: ""
    default property alias trailing: trailingRow.data

    anchors { top: parent.top; left: parent.left; right: parent.right }
    height: pal.headerH

    Rectangle {
        anchors.fill: parent
        color: pal.bg

        Rectangle {
            anchors { left: parent.left; right: parent.right; bottom: parent.bottom }
            height: 1
            color: pal.border
        }
    }

    RIconButton {
        id: backBtn
        visible: hdr.back
        width: visible ? units.gu(4.5) : 0
        anchors { left: parent.left; leftMargin: units.gu(0.5); verticalCenter: parent.verticalCenter }
        name: "chevronLeft"
        tint: pal.text
        onClicked: stack.pop()
    }

    RIcon {
        id: leadIcon
        visible: hdr.icon !== ""
        width: visible ? units.gu(3) : 0
        anchors {
            left: backBtn.right
            leftMargin: visible ? units.gu(1) : 0
            verticalCenter: parent.verticalCenter
        }
        name: hdr.icon
        tint: pal.accent
        size: units.gu(3)
    }

    Text {
        anchors {
            left: leadIcon.right
            leftMargin: units.gu(1.5)
            right: trailingRow.left
            rightMargin: units.gu(1)
            verticalCenter: parent.verticalCenter
        }
        text: hdr.title
        color: pal.text
        font.pixelSize: pal.fsTitle
        font.weight: Font.DemiBold
        elide: Text.ElideRight
    }

    Row {
        id: trailingRow
        anchors { right: parent.right; rightMargin: units.gu(0.5); verticalCenter: parent.verticalCenter }
        spacing: units.gu(0.25)
    }
}
