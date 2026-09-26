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
- **готово.** Manager — сесія в додатку провайдера (Codex, Claude або Gemini), з коренем data root. Fleet створює її на першому запуску і пам’ятає identity. Розмова живе в додатку провайдера. Cursor як Manager — немає. Push у чужу сесію не будувати.

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

## Порядок

1. **немає.** Локальний API на `127.0.0.1:8787` не перевіряє `Host` і `Origin` і не має токена запуску. Будь-яка сторінка в браузері може звернутися до API, який запускає агентів і працює з файлами. Потрібно: allowlist для `Host`, перевірка `Origin`, токен запуску, відмова від тіл не-JSON. Це стосується і dashboard, і Manager MCP на тому самому порту. Публічний бінарник цим блокується.
2. **готово.** Шина подій. Канал `task`: `task.needs_attention`, `task.needs_review`. Зроблено раніше за локальний API. Інші канали без видавців. Публічний бінарник це не відчиняє.
3. **немає.** Збірка з [release](release.md): UI в бінарнику, версія з тега, workflow публікує `darwin/arm64` і checksums у Site.
4. **поза релізом.** [Windows](../specs/windows.md). Асет не публікувати.
5. Те, що не входить у випуск, зібрано в [release](release.md).
