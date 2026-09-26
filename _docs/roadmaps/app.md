# App

Fleet координує Codex, Claude Code, Cursor і людину. Fleet не є ще одним coding agent. Manager — не модель всередині Fleet. Це виділена сесія агента, якого людина вже використовує.

## Цикл

**частково.** Прямий хід (задача, worker, `needs_review`) у App є. Подія `task.needs_attention` і `task.needs_review` доходить до прив’язаної сесії Manager. Відповідь людини в тій сесії і подальше продовження worker-а цим каналом не вимірюються.

```text
rough human intent
        ↓
     Manager
        ↓
structured task / dependencies
        ↓
     Fleet
        ↓
native worker
        ↓
Running
        ↓
Needs Attention / Needs Review
        ↓
     Manager
        ↓
      Human
        ↓
worker continues
        ↓
review + Accepted
```

Manager бачить проєкти, задачі, залежності, статуси, workers, blockers, ownership і review state через Fleet/MCP. Доступ до filesystem йому не потрібен. Сховище стану (markdown сьогодні, інше завтра) для Manager невидиме.

Якщо задача ще не готова до імплементації, Manager створює окрему research-задачу, яка блокує імплементацію. Специфікацію він не вигадує.

- **готово.** Manager API і MCP на локальному сервері.
- **готово.** Статус `needs_review` і продовження роботи після нього.
- **готово.** Шина подій. Канал `task` публікує `task.needs_attention`, коли задача стає `blocked` (запуск не вдався, timeout, orphan) і коли worker ставить питання: пауза лишає статус `doing`. `task.needs_review` — коли робота готова до review. Канали `release`, `worker`, `practice`, `configuration` названі, видавців немає. Подія лишається на шині. У сесію Manager її ніхто не відправляє. Візуалізація стрічки — наступний крок.
- **готово.** Manager — сесія Claude, Codex, Cursor або Gemini з коренем data root. Для Claude, Codex і Gemini Fleet створює сесію і пам’ятає identity. Для Cursor Fleet не запускає процес: Manager — це ця папка, відкрита в Cursor. З телефона Cursor-сесія не видна. Розмова живе в додатку провайдера. Push у чужу сесію не будувати.

`send_message_to_thread` з bundled `codex-app-tools` не є транспортом Fleet. Інструмент живе всередині Codex Desktop. Зовнішній процес Fleet до цього pipe не підключений. Inbox поверх нього не будувати, і в сесію Manager Fleet не пише.

Manager бачить дошку через Fleet MCP у рідному конфігу свого провайдера. Людина відповідає в додатку провайдера. Manager через MCP оновлює задачу, worker продовжує.

Пізніше той самий `Publish`, без нової схеми доставки: `worker.offline`, `worker.quota_low`, `release.available`, `practice.available`, `configuration.recommended`. У цей випуск вони не входять.

## Ownership

**готово** як інваріант. Один агент, один репозиторій, одна активна задача. Різні репозиторії йдуть паралельно. Один репозиторій серіалізований.

```text
Running         → ownership HELD
Needs Attention → ownership HELD
Needs Review    → ownership HELD
Accepted        → ownership RELEASED
```

Blocked і Needs Attention не є завершенням. Зупинена сесія не є прийнятою роботою. Наступний worker отримує репозиторій лише після Accepted.

## Activation

**немає** як вимірювана подія. Подія задачі є на шині, але в сесію Manager не доставляється. Коло для сторонньої людини все ще не замкнене, поки людина не відповідає в додатку провайдера і worker не їде далі до Accepted.

## Manager: skills, код і MCP

**немає.** Спека: [manager-skills](../specs/manager-skills.md).

Код тримає одну правильну відповідь (ID, слаги, статуси, дедуп, валідація). MCP віддає компактні факти і приймає атомарні записи. Skill тримає лише судження: коли питати, як різати задачу, що вважати готовим. Не більше семи skills: `intake`, `shape-task`, `resolve-project`, `route`, `triage-attention`, `review`, `briefing`. Джерело живе в App; provisioning пише їх разом із MCP-конфігами.

Зараз Manager на старті читає великі документи data root і може відкрити `Work/INDEX.md` або `_registry/repositories.json`. Більшість тих правил уже виконує код.

Події в сесію не пушаться. `manager_events` лише читає шину за курсором.

1. **немає.** `manager_board`, `manager_task` і skill `intake`.
2. **немає.** Поля `depends_on`, `repositories`, `acceptance_criteria` в `Intent` і `manager_validate`.
3. **немає.** `manager_resolve_project`, `manager_similar`, `manager_route`. Маршрут стає конфігом, не прозою `ROUTING.md`.
4. **немає.** `manager_inbox`, `manager_project`.
5. **немає.** `manager_answer`, `manager_review`, `manager_decompose` і skills `triage-attention`, `review`.

Захист порту для цих записів уже є, див. крок 1 нижче.

## Порядок

1. **готово.** Локальний API на `127.0.0.1:8787` приймає лише `Host` `127.0.0.1`, `localhost` і `[::1]` зі своїм портом, відхиляє чужий `Origin`, вимагає токен запуску на `/api` і `/mcp` і не бере тіло не-JSON. Порожній POST і аудіо `multipart` лишаються. Токен один на інсталяцію: лежить у `~/.fleet/launch-token` (0600), створюється раз і переживає перезапуск, тож відкриті вкладки й сесії Manager не ламаються. У консоль він не друкується. Дашборд читає його з `<meta>` (сторінка йде з `no-store`), Fleet MCP отримує його в URL, тому в файлах конфігів провайдерів (`.mcp.json`, `.codex/config.toml`, `.cursor/mcp.json`, `.agents/mcp_config.json`) лежить цей токен. Це частина сетапу, файли трекаються. Токен стабільний, тож рядок міняється один раз, а не при кожному запуску. Він дійсний лише для `127.0.0.1` зі своїм Host, але в query потрапляє в логи, тож URL не публікувати. У dev `vite` проксі сам додає токен і Host. `POST /api/manager/session` і `POST /api/manager/adopt` під цим самим захистом: вони пишуть Fleet MCP і для Codex довіряють теку в глобальному `~/.codex/config.toml`.
2. **готово.** Шина подій. Канал `task`: `task.needs_attention`, `task.needs_review`. Зроблено раніше за локальний API. Інші канали без видавців. Публічний бінарник це не відчиняє.
3. **немає.** Збірка з [release](release.md): UI в бінарнику, версія з тега, workflow публікує `darwin/arm64` і checksums у цей самий репозиторій App.
4. **поза релізом.** [Windows](../specs/windows.md). Асет не публікувати.
5. Те, що не входить у випуск, зібрано в [release](release.md).

## Відоме, не в цьому випуску

**частково.** Запис Fleet MCP: невідомі ключі й чужі сервери зберігаються, битий JSON не перезаписується (помилка), перша копія файлу лишається як `*.fleet-bak`. Секцію TOML для Codex Fleet замінює власним редактором, а не парсером.

**немає.** Тести пакета `internal/server` не проходять з `-count>1`: кілька тестів запуску чекають до 5 с і падають на другому проході, тож між ітераціями лишається спільний стан. Один прохід зелений. У повному `go test ./...` під навантаженням один такий тест раз упав.
