import QtQuick 2.7
import Ubuntu.Components 1.3

// Строка списка-карточки: значок, заголовок с подписью, справа значение,
// кнопки (default-свойство) и шеврон. Текстовый блок упирается в левую
// грань блока действий, поэтому длинные имена не выносят кнопки за экран.
Rectangle {
    id: row

    property string icon: ""
    property color tint: pal.accent
    property string title: ""
    property string subtitle: ""
    property string value: ""
    property color valueColor: pal.dim
    property bool active: false
    property bool chevron: false
    default property alias actions: actRow.data
    signal clicked()

    height: units.gu(8.5)
    radius: pal.radius
    color: active ? Qt.rgba(pal.accent.r, pal.accent.g, pal.accent.b, 0.10) : pal.surface
    border.width: 1
    border.color: active ? Qt.rgba(pal.accent.r, pal.accent.g, pal.accent.b, 0.55) : pal.border

    MouseArea {
        id: ma
        anchors.fill: parent
        onClicked: row.clicked()
    }

    Rectangle {
        id: badge
        visible: row.icon !== ""
        width: visible ? units.gu(4.5) : 0
        height: units.gu(4.5)
        radius: units.gu(1.2)
        anchors { left: parent.left; leftMargin: units.gu(1.5); verticalCenter: parent.verticalCenter }
        color: Qt.rgba(row.tint.r, row.tint.g, row.tint.b, 0.16)
        opacity: ma.pressed ? 0.6 : 1

        RIcon {
            anchors.centerIn: parent
            name: row.icon
            tint: row.tint
            size: units.gu(2.4)
        }
    }

    Item {
        id: textBlock
        anchors {
            left: badge.right
            leftMargin: units.gu(1.5)
            right: tail.left
            rightMargin: units.gu(1)
            verticalCenter: parent.verticalCenter
        }
        // Высота считается от строк, а не childrenRect: у якорёванных детей
        // childrenRect даёт цикл привязок.
        height: titleText.height + (subText.visible ? subText.height + units.gu(0.25) : 0)

        Text {
            id: titleText
            anchors { left: parent.left; right: parent.right; top: parent.top }
            text: row.title
            color: pal.text
            font.pixelSize: pal.fsBody
            font.weight: Font.DemiBold
            elide: Text.ElideRight
        }
        Text {
            id: subText
            anchors { left: parent.left; right: parent.right; top: titleText.bottom; topMargin: units.gu(0.25) }
            visible: row.subtitle !== ""
            text: row.subtitle
            color: pal.dim
            font.pixelSize: pal.fsSmall
            elide: Text.ElideRight
        }
    }

    Row {
        id: tail
        anchors { right: parent.right; rightMargin: units.gu(1); verticalCenter: parent.verticalCenter }
        spacing: units.gu(0.5)

        Text {
            visible: row.value !== ""
            anchors.verticalCenter: parent.verticalCenter
            text: row.value
            color: row.valueColor
            font.pixelSize: pal.fsSmall
        }
        Row {
            id: actRow
            anchors.verticalCenter: parent.verticalCenter
            spacing: units.gu(0.25)
        }
        RIcon {
            visible: row.chevron
            width: visible ? units.gu(2) : 0
            anchors.verticalCenter: parent.verticalCenter
            name: "chevronRight"
            tint: pal.faint
            size: units.gu(2)
        }
    }
}
