import QtQuick 2.7
import Ubuntu.Components 1.3

// Подпись над полем (section: false) или заголовок секции (section: true).
Text {
    property bool section: false

    color: pal.dim
    font.pixelSize: section ? pal.fsTiny : pal.fsSmall
    font.weight: section ? Font.DemiBold : Font.Normal
    font.capitalization: section ? Font.AllUppercase : Font.MixedCase
    font.letterSpacing: section ? units.dp(0.4) : 0
    elide: Text.ElideRight
}
