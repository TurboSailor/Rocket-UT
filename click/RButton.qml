import QtQuick 2.7
import Ubuntu.Components 1.3

// Кнопка. variant: primary | success | danger | ghost | plain.
// Ширину задаёт вызывающий (обычно parent.width или доля от него).
Rectangle {
    id: btn

    property string text: ""
    property string icon: ""
    property string variant: "ghost"
    signal clicked()

    readonly property bool solid: variant === "primary" || variant === "success" || variant === "danger"
    readonly property color tone: variant === "primary" ? pal.accent
                                : variant === "success" ? pal.ok
                                : variant === "danger" ? pal.bad
                                : pal.surfaceAlt

    height: pal.ctlH
    radius: pal.radiusSm
    color: variant === "plain" ? "transparent" : tone
    border.width: variant === "ghost" || variant === "plain" ? 1 : 0
    border.color: pal.border
    opacity: btn.enabled ? (ma.pressed ? 0.7 : 1) : 0.35

    Row {
        anchors.centerIn: parent
        spacing: units.gu(1)

        RIcon {
            visible: btn.icon !== ""
            width: visible ? units.gu(2.2) : 0
            anchors.verticalCenter: parent.verticalCenter
            name: btn.icon
            size: units.gu(2.2)
            tint: btn.solid ? "#FFFFFF" : pal.text
        }
        Text {
            anchors.verticalCenter: parent.verticalCenter
            text: btn.text
            color: btn.solid ? "#FFFFFF" : pal.text
            font.pixelSize: pal.fsBody
            font.weight: Font.DemiBold
            elide: Text.ElideRight
            width: Math.min(implicitWidth, btn.width - units.gu(4))
            horizontalAlignment: Text.AlignHCenter
        }
    }

    MouseArea {
        id: ma
        anchors.fill: parent
        enabled: btn.enabled
        onClicked: btn.clicked()
    }
}
