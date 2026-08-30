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

    header: PageHeader {
        id: hdr
        title: root.tr("Scan QR")
        trailingActionBar.actions: [
            Action {
                iconName: page.macro ? "zoom-in" : "zoom-out"
                text: root.tr("Macro focus")
                onTriggered: page.setMacro(!page.macro)
            },
            Action {
                iconName: "rotate-right"
                text: root.tr("Rotate preview")
                onTriggered: root.qrPreviewRotation = (root.qrPreviewRotation + 90) % 360
            },
            Action {
                iconName: "camera-flip"
                text: root.tr("Flip camera")
                visible: QtMultimedia.availableCameras.length > 1
                onTriggered: page.flipCamera()
            }
        ]
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
        // autoOrientation — проверенный на этом устройстве путь; ручная поправка
        // остаётся как страховка, если платформа определит поворот неверно.
        autoOrientation: root.qrPreviewRotation === 0
        orientation: root.qrPreviewRotation

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
            border.color: cam.lockStatus === Camera.Locked ? UbuntuColors.green
                        : (cam.lockStatus === Camera.Searching ? UbuntuColors.orange
                                                               : UbuntuColors.silk)
            opacity: 0.9
        }

        ActivityIndicator {
            anchors { top: parent.top; right: parent.right; margins: units.gu(2) }
            running: page.busy
        }
    }

    Column {
        id: panel
        anchors { bottom: parent.bottom; left: parent.left; right: parent.right; margins: units.gu(2) }
        spacing: units.gu(1)

        Label {
            id: hint
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            fontSize: "small"
            text: root.tr("point the camera at a QR code")
        }
        Label {
            id: msg
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
            wrapMode: Text.Wrap
            fontSize: "small"
            color: UbuntuColors.red
        }
        Label {
            width: parent.width
            wrapMode: Text.WrapAnywhere
            fontSize: "x-small"
            visible: page.result !== ""
            text: page.result
        }

        Row {
            width: parent.width
            spacing: units.gu(1)
            visible: page.result === ""

            Button {
                width: (parent.width - units.gu(1)) / 2
                text: root.tr("Capture")
                color: UbuntuColors.blue
                enabled: !page.busy && cam.cameraStatus === Camera.ActiveStatus
                onClicked: page.shoot(true)
            }
            Button {
                width: (parent.width - units.gu(1)) / 2
                text: page.autoScan ? root.tr("Auto: on") : root.tr("Auto: off")
                onClicked: page.autoScan = !page.autoScan
            }
        }

        Button {
            width: parent.width
            text: root.tr("Import as node")
            color: UbuntuColors.green
            visible: page.result !== ""
            onClicked: page.apply("node")
        }
        Button {
            width: parent.width
            text: root.tr("Add as subscription")
            visible: page.result !== "" && page.looksLikeURL(page.result)
            onClicked: page.apply("sub")
        }
        Button {
            width: parent.width
            text: root.tr("Scan again")
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
