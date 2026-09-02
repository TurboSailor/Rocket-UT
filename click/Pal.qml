import QtQuick 2.7
import Ubuntu.Components 1.3

// Палитра и метрики. Светлая тема — по умолчанию, тёмная включается
// свойством dark (его хранит main.qml и запоминает между запусками).
// Все размеры — units.gu/units.dp: сырые пиксели на 400+ DPI нечитаемы.
QtObject {
    property bool dark: false

    readonly property color bg:         dark ? "#0B0E14" : "#F3F5F8"
    readonly property color surface:    dark ? "#161C27" : "#FFFFFF"
    readonly property color surfaceAlt:  dark ? "#1E2635" : "#E9ECF2"
    readonly property color border:     dark ? "#27303F" : "#D5DAE3"
    readonly property color text:       dark ? "#EBEFF7" : "#151922"
    readonly property color dim:        dark ? "#8A94A8" : "#5E687A"
    readonly property color faint:      dark ? "#5C6577" : "#98A1B2"
    readonly property color accent:     dark ? "#3D7BFF" : "#2563EB"
    readonly property color ok:         dark ? "#27C46B" : "#12864A"
    readonly property color warn:       dark ? "#F5A524" : "#B26A00"
    readonly property color bad:        dark ? "#F0464B" : "#CE2B30"
    readonly property color violet:     dark ? "#8B5CF6" : "#6D28D9"

    readonly property real radius:   units.gu(1.5)
    readonly property real radiusSm: units.gu(1)
    readonly property real pad:      units.gu(2)
    readonly property real gap:      units.gu(1.5)
    readonly property real ctlH:     units.gu(6)
    readonly property real fieldH:   units.gu(5.5)
    readonly property real headerH:  units.gu(7)

    readonly property real fsTitle: units.dp(19)
    readonly property real fsBody:  units.dp(15)
    readonly property real fsSmall: units.dp(13)
    readonly property real fsTiny:  units.dp(11)
    readonly property real stroke:  units.dp(0.8)
}
