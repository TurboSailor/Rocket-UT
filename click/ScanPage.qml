import QtQuick 2.7
import QtMultimedia 5.8
import Ubuntu.Components 1.3

// Сканер QR.
//
// Резкость: на этом устройстве поддерживаются FocusContinuous и FocusMacro
// (проверено запросом isFocusModeSupported). QR держат близко, поэтому есть
// переключатель «Макро», точка фокуса по умолчанию — центр кадра, а тап по
// превью наводит фокус в выбранную точку через FocusPointCustom + searchAndLock.
// Рамка меняет цвет по lockStatus, чтобы видеть, поймал ли автофокус резкость.
//
// Частота: полноценный снимок (imageCapture) прерывает видоискатель и автофокус,
// поэтому большинство тактов — быстрый и бесшумный grabToImage превью, а реальный
// снимок делается лишь каждый 4-й такт (плотный QR в превью может не читаться).
// Пока запрос к демону не завершён, новые такты пропускаются.
Page {
    id: page

    property string result: ""
    property bool busy: false
    property double busyAt: 0
    property int shots: 0
    property bool autoScan: true
    property bool macro: false

    // Датчик установлен повёрнутым: на этом устройстве задняя камера сообщает
    // orientation=270, передняя 90. Без компенсации превью лежит на боку.
    readonly property int sensorFix: cam.position === Camera.FrontFace
        ? (cam.orientation + 360) % 360
        : (360 - cam.orientation) % 360

    Rectangle {
        anchors.fill: parent
        color: pal.bg
    }

    RHeader {
        id: hdr
        title: root.tr("Scan QR")

        RIconButton {
            name: page.macro ? "zoomIn" : "zoomOut"
            tint: page.macro ? pal.accent : pal.text
            onClicked: page.setMacro(!page.macro)
        }
        RIconButton {
            name: "rotate"
            tint: pal.text
            onClicked: root.qrPreviewRotation = (root.qrPreviewRotation + 90) % 360
        }
        RIconButton {
            name: "flip"
            tint: pal.text
            visible: QtMultimedia.availableCameras.length > 1
            onClicked: page.flipCamera()
        }
    }

    function setMacro(on) {
        page.macro = on
        var want = on ? Camera.FocusMacro : Camera.FocusContinuous
        if (cam.focus.isFocusModeSupported(want))
            cam.focus.focusMode = want
        else if (cam.focus.isFocusModeSupported(Camera.FocusAuto))
            cam.focus.focusMode = Camera.FocusAuto
        page.refocus()
    }

    // refocus просит автофокус заново поискать резкость.
    function refocus() {
        if (cam.cameraStatus !== Camera.ActiveStatus)
            return
        cam.unlock()
        cam.searchAndLock()
    }

    function focusAt(x, y) {
        if (cam.focus.isFocusPointModeSupported(Camera.FocusPointCustom)) {
            cam.focus.customFocusPoint = Qt.point(Math.max(0, Math.min(1, x)),
                                                  Math.max(0, Math.min(1, y)))
            cam.focus.focusPointMode = Camera.FocusPointCustom
        }
        page.refocus()
    }

    function flipCamera() {
        var cams = QtMultimedia.availableCameras
        for (var i = 0; i < cams.length; i++) {
            if (cams[i].deviceId !== cam.deviceId) {
                cam.deviceId = cams[i].deviceId
                break
            }
        }
        page.applyFocusDefaults()
    }

    function applyFocusDefaults() {
        if (cam.focus.isFocusPointModeSupported(Camera.FocusPointCenter))
            cam.focus.focusPointMode = Camera.FocusPointCenter
        page.setMacro(page.macro)
    }

    onVisibleChanged: {
        if (visible) {
            page.result = ""
            page.busy = false
            page.shots = 0
            hint.text = root.tr("point the camera at a QR code")
            msg.text = ""
        }
    }

    Camera {
        id: cam
        captureMode: Camera.CaptureStillImage
        // Жизненный цикл привязан к странице: камера не держится вне сканера.
        cameraState: page.visible ? Camera.ActiveState : Camera.UnloadedState
        onCameraStatusChanged: if (cameraStatus === Camera.ActiveStatus) page.applyFocusDefaults()
        onError: msg.text = root.tr("camera") + ": " + errorString

        imageCapture {
            // 4:3 под сенсор: 16:9 давало обрезку и предупреждение о aspect ratio.
            resolution: Qt.size(1600, 1200)
            onImageSaved: page.decode(path)
            onCaptureFailed: {
                page.busy = false
                msg.text = root.tr("capture failed")
            }
        }
    }

    VideoOutput {
        id: view
        anchors { top: hdr.bottom; left: parent.left; right: parent.right; bottom: panel.top }
        source: cam
        fillMode: VideoOutput.PreserveAspectCrop
        // autoOrientation на этом устройстве определяет поворот неверно
        // (кадр стартует лежащим на боку), поэтому считаем сами:
        // sensorFix компенсирует установку датчика, qrPreviewRotation — правка
        // пользователя, она запоминается между запусками.
        autoOrientation: false
        orientation: (page.sensorFix + root.qrPreviewRotation) % 360

        MouseArea {
            anchors.fill: parent
            onClicked: {
                page.focusAt(mouse.x / width, mouse.y / height)
                hint.text = root.tr("focusing…")
            }
        }

        // Рамка прицела: цвет показывает состояние автофокуса.
        Rectangle {
            anchors.centerIn: parent
            width: Math.min(parent.width, parent.height) * 0.7
            height: width
            color: "transparent"
            radius: units.gu(1)
            border.width: units.dp(2)
            border.color: cam.lockStatus === Camera.Locked ? pal.ok
                        : (cam.lockStatus === Camera.Searching ? pal.warn
                           : Qt.rgba(pal.text.r, pal.text.g, pal.text.b, 0.55))
            opacity: 0.9
        }

        ActivityIndicator {
            anchors { top: parent.top; right: parent.right; margins: units.gu(2) }
            running: page.busy
        }
    }

    // Нижняя панель: полупрозрачная подложка поверх кадра, скруглённая сверху
    // (нижние углы уводятся за край экрана отрицательным отступом).
    Rectangle {
        id: panel
        anchors {
            bottom: parent.bottom
            bottomMargin: -pal.radius
            left: parent.left
            right: parent.right
        }
        height: pcol.height + 2 * pal.pad + pal.radius
        radius: pal.radius
        color: Qt.rgba(pal.bg.r, pal.bg.g, pal.bg.b, 0.92)
        border.width: 1
        border.color: pal.border

        Column {
            id: pcol
            anchors {
                top: parent.top; topMargin: pal.pad
                left: parent.left; leftMargin: pal.pad
                right: parent.right; rightMargin: pal.pad
            }
            spacing: units.gu(1)

            Text {
                id: hint
                width: parent.width
                horizontalAlignment: Text.AlignHCenter
                wrapMode: Text.Wrap
                color: pal.dim
                font.pixelSize: pal.fsSmall
                text: root.tr("point the camera at a QR code")
            }

            RNote {
                id: msg
                width: parent.width
                tone: "bad"
            }

            Text {
                width: parent.width
                visible: page.result !== ""
                text: page.result
                color: pal.text
                font.family: "Ubuntu Mono"
                font.pixelSize: pal.fsSmall
                wrapMode: Text.WrapAnywhere
                maximumLineCount: 3
                elide: Text.ElideRight
            }

            Row {
                width: parent.width
                spacing: units.gu(1)
                visible: page.result === ""

                RButton {
                    width: (parent.width - units.gu(1)) / 2
                    icon: "camera"
                    text: root.tr("Capture")
                    variant: "primary"
                    enabled: !page.busy && cam.cameraStatus === Camera.ActiveStatus
                    onClicked: page.shoot(true)
                }
                RButton {
                    width: (parent.width - units.gu(1)) / 2
                    variant: "ghost"
                    text: page.autoScan ? root.tr("Auto: on") : root.tr("Auto: off")
                    onClicked: page.autoScan = !page.autoScan
                }
            }

            RButton {
                width: parent.width
                icon: "download"
                text: root.tr("Import as node")
                variant: "success"
                visible: page.result !== ""
                onClicked: page.apply("node")
            }
            RButton {
                width: parent.width
                icon: "link"
                text: root.tr("Add as subscription")
                variant: "ghost"
                visible: page.result !== "" && page.looksLikeURL(page.result)
                onClicked: page.apply("sub")
            }
            RButton {
                width: parent.width
                text: root.tr("Scan again")
                variant: "plain"
                visible: page.result !== ""
                onClicked: {
                    page.result = ""
                    page.busy = false
                    page.shots = 0
                    hint.text = root.tr("point the camera at a QR code")
                    page.refocus()
                }
            }
        }
    }

    // shoot: full=true — реальный снимок (нужен для плотных QR),
    // иначе бесшумный захват превью, который не сбивает автофокус.
    function shoot(full) {
        if (page.busy) {
            // Страховка от подвисшего запроса.
            if ((Date.now() - page.busyAt) < 2500)
                return
        }
        page.busy = true
        page.busyAt = Date.now()
        if (full) {
            if (cam.imageCapture.ready)
                cam.imageCapture.captureToLocation("/tmp/rocket-qr-frame.png")
            else
                page.busy = false
            return
        }
        view.grabToImage(function(res) {
            if (!res) { page.busy = false; return }
            res.saveToFile("/tmp/rocket-qr-frame.png")
            page.decode("/tmp/rocket-qr-frame.png")
        })
    }

    function decode(path) {
        root.api("/qrdecode", function(r, code) {
            page.busy = false
            if (r && r.text) {
                page.result = r.text
                page.autoScan = false
                hint.text = root.tr("QR found")
                msg.text = ""
                return
            }
            // 422 — в кадре нет QR, это нормальный ход поиска.
            if (code !== 422 && r && r.error)
                msg.text = r.error
        }, "POST", path)
    }

    Timer {
        interval: 900
        repeat: true
        running: page.visible && page.autoScan && page.result === ""
                 && cam.cameraStatus === Camera.ActiveStatus
        onTriggered: {
            page.shots += 1
            // Каждый 4-й такт — реальный снимок: превью может не хватить
            // разрешения для плотного QR (например, конфига AmneziaWG).
            page.shoot(page.shots % 4 === 0)
            if (page.shots % 6 === 0 && page.result === "")
                hint.text = root.tr("looking for QR…")
        }
    }

    // Периодический повтор автофокуса: непрерывный AF на этом стеке
    // иногда «залипает» и без нового поиска резкость не появляется.
    Timer {
        interval: 3000
        repeat: true
        running: page.visible && page.result === "" && !page.macro
                 && cam.cameraStatus === Camera.ActiveStatus
        onTriggered: if (cam.lockStatus !== Camera.Locked) page.refocus()
    }

    // looksLikeURL: подписка — это http(s)-ссылка, но http:// бывает и прокси-узлом,
    // поэтому выбор всегда за пользователем, а не за эвристикой.
    function looksLikeURL(s) {
        // sub:// и shadowrocket://subscribe — однозначно подписки;
        // http(s) может быть и подпиской, и прокси-узлом, поэтому решает пользователь.
        return s.indexOf("http://") === 0 || s.indexOf("https://") === 0
            || s.indexOf("sub://") === 0 || s.indexOf("shadowrocket://") === 0
    }

    function apply(as) {
        msg.text = root.tr("working…")
        if (as === "sub") {
            root.api("/subs/add?name=" + encodeURIComponent(root.tr("QR")) +
                     "&url=" + encodeURIComponent(page.result),
                function(r, code) {
                    if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
                    root.absorb(r); root.refresh(); stack.pop()
                }, "POST")
            return
        }
        root.api("/nodes/import?name=awg", function(r, code) {
            if (!r || r.error) { msg.text = (r && r.error) || ("HTTP " + code); return }
            root.absorb(r); root.refresh(); stack.pop()
        }, "POST", page.result)
    }
}
