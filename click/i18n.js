.pragma library

var lang = "en"

function setLang(l) {
    lang = (String(l).substring(0, 2).toLowerCase() === "ru") ? "ru" : "en"
    return lang
}

var D = {
    "Rocket":                 {en: "Rocket",               ru: "Rocket"},
    "CONNECTED":              {en: "CONNECTED",            ru: "ПОДКЛЮЧЕНО"},
    "DISCONNECTED":           {en: "DISCONNECTED",         ru: "ОТКЛЮЧЕНО"},
    "daemon unavailable":     {en: "daemon unavailable",   ru: "демон недоступен"},
    "no node selected":       {en: "no node selected",     ru: "узел не выбран"},
    "Nodes":                  {en: "Nodes",                ru: "Узлы"},
    "Subscriptions":          {en: "Subscriptions",        ru: "Подписки"},
    "Configs":                {en: "Configs",              ru: "Конфиги"},
    "Log":                    {en: "Log",                  ru: "Журнал"},
    "Import":                 {en: "Import",               ru: "Импорт"},
    "Routing":                {en: "Routing",              ru: "Маршрутизация"},
    "Direct":                 {en: "Direct",               ru: "Напрямую"},
    "Proxy":                  {en: "Proxy",                ru: "Прокси"},
    "Sections":               {en: "Sections",             ru: "Разделы"},
    "Basics":                 {en: "Basics",               ru: "Основное"},
    "Access":                 {en: "Access",               ru: "Доступ"},
    "Transport":              {en: "Transport",            ru: "Транспорт"},
    "node unreachable":       {en: "node unreachable",     ru: "узел недоступен"},
    "Add":                    {en: "Add",                  ru: "Добавить"},
    "Save":                   {en: "Save",                 ru: "Сохранить"},
    "Delete":                 {en: "Delete",               ru: "Удалить"},
    "Cancel":                 {en: "Cancel",               ru: "Отмена"},
    "Name":                   {en: "Name",                 ru: "Имя"},
    "URL":                    {en: "URL",                  ru: "Ссылка"},
    "Paste links or config":  {en: "Paste links or config", ru: "Вставьте ссылки или конфиг"},
    "Open file":              {en: "Open file",            ru: "Открыть файл"},
    "no nodes yet":           {en: "no nodes yet",         ru: "узлов пока нет"},
    "no subscriptions yet":   {en: "no subscriptions yet",  ru: "подписок пока нет"},
    "no configs yet":         {en: "no configs yet",        ru: "конфигов пока нет"},
    "no traffic yet":         {en: "no traffic yet",        ru: "трафика пока нет"},
    "Add rule":               {en: "Add rule",             ru: "Добавить правило"},
    "Rule type":              {en: "Rule type",            ru: "Тип правила"},
    "Value":                  {en: "Value",                ru: "Значение"},
    "Policy":                 {en: "Policy",               ru: "Политика"},
    "Add from URL":           {en: "Add from URL",          ru: "Добавить по ссылке"},
    "unsupported lines":      {en: "unsupported lines",     ru: "неподдерж. строк"},
    "rules":                  {en: "rules",                ru: "правил"},
    "proxies":                {en: "proxies",              ru: "прокси"},
    "active":                 {en: "active",               ru: "активен"},
    "nodes":                  {en: "nodes",                ru: "узлов"},
    "never":                  {en: "never",                ru: "никогда"},
    "testing…":               {en: "testing…",             ru: "проверка…"},
    "working…":               {en: "working…",             ru: "выполняется…"},
    "Config name":            {en: "Config name",          ru: "Имя конфига"},
    "AmneziaWG name":         {en: "AmneziaWG name",       ru: "Имя AmneziaWG"},
    "Edit config":            {en: "Edit config",          ru: "Правка конфига"},
    "Edit node":              {en: "Edit node",            ru: "Правка узла"},
    "Test":                   {en: "Test",                 ru: "Проверить"},
    "Server":                 {en: "Server",               ru: "Сервер"},
    "Port":                   {en: "Port",                 ru: "Порт"},
    "Username":               {en: "Username",             ru: "Логин"},
    "Password":               {en: "Password",             ru: "Пароль"},
    "Method":                 {en: "Method",               ru: "Метод"},
    "Path":                   {en: "Path",                 ru: "Путь"},
    "Host header":            {en: "Host header",           ru: "Заголовок Host"},
    "Skip cert check":        {en: "Skip cert check",       ru: "Не проверять сертификат"},
    "from subscription":      {en: "from subscription",     ru: "из подписки"},
    "editing detaches the node from its subscription": {
        en: "editing detaches the node from its subscription",
        ru: "правка отвязывает узел от подписки"
    },
    "Rules":                  {en: "Rules",                ru: "Правила"},
    "no rules yet":           {en: "no rules yet",          ru: "правил пока нет"},
    "Edit rule":              {en: "Edit rule",            ru: "Правка правила"},
    "Scan QR":                {en: "Scan QR",              ru: "Сканировать QR"},
    "QR from image":          {en: "QR from image",         ru: "QR из картинки"},
    "Capture":                {en: "Capture",              ru: "Снять кадр"},
    "Auto: on":               {en: "Auto: on",             ru: "Авто: вкл"},
    "Auto: off":              {en: "Auto: off",            ru: "Авто: выкл"},
    "focusing…":              {en: "focusing…",            ru: "наводка резкости…"},
    "looking for QR…":        {en: "looking for QR…",       ru: "поиск QR…"},
    "QR found":               {en: "QR found",             ru: "QR распознан"},
    "camera":                 {en: "camera",               ru: "камера"},
    "capture failed":         {en: "capture failed",        ru: "снимок не удался"},
    "QR":                     {en: "QR",                   ru: "QR"},
    "Import as node":         {en: "Import as node",        ru: "Импорт как узел"},
    "Add as subscription":    {en: "Add as subscription",    ru: "Добавить как подписку"},
    "Scan again":             {en: "Scan again",            ru: "Сканировать снова"},
    "point the camera at a QR code": {
        en: "point the camera at a QR code",
        ru: "наведите камеру на QR-код"
    },
    "Import from file or paste text below": {
        en: "Import from file or paste text below",
        ru: "Импортируйте из файла или вставьте текст ниже"
    },
    "Supported: vless, vmess, ss, trojan, socks5, http, ssh URIs, sub:// links, Shadowrocket .conf, AmneziaWG .conf": {
        en: "Supported: vless, vmess, ss, trojan, socks5, http, ssh URIs, sub:// links, Shadowrocket .conf, AmneziaWG .conf",
        ru: "Поддерживается: ссылки vless, vmess, ss, trojan, socks5, http, ssh, ссылки sub://, .conf Shadowrocket, .conf AmneziaWG"
    }
}

function tr(k) {
    var e = D[k]
    if (!e)
        return k
    return e[lang] || e.en || k
}
