# Docs (App)

Документація застосунку Fleet: `server`, `view` і `site`. Документи про
Inbox/Work/Fleet/Archive для оператора лежать у `_docs/` його data root (їх
початковий вигляд — `server/internal/appconfig/template`), не тут.

Тут лише актуальний стан. Історія змін лежить у git, плани, яких немає в коді,
у роадмапах.

Позначки в роадмапах і ТЗ:

- **готово** — є в репозиторії зараз
- **частково** — шматок є, решта відсутня
- **немає** — описано, в репозиторії цього немає
- **поза релізом** — свідомо не зараз

Роадмап ставить пункт у порядок і лінкує спеку. Спека — єдине місце, де розписано, що саме робити.

Роадмапи й спеки українською з позначками статусу. Технічні документи англійською, без позначок статусу: вони описують те, що є.

## Роадмапи

- [App](roadmaps/app.md) — застосунок Fleet і його цикл
- [Site](roadmaps/site.md) — публічний сайт і лендінг
- [Release](roadmaps/release.md) — порядок випуску, контракт дистрибуції і те, що не входить

## Спеки

- [Windows](specs/windows.md)
- [Manager: skills, код і MCP](specs/manager-skills.md)

## Технічні документи

- [ARCHITECTURE](ARCHITECTURE.md) — як влаштований Fleet: запис задач, контури, запуск, сховище, модель збоїв, пакети.
- [DOMAIN_MODEL](DOMAIN_MODEL.md) — workspace, project, repository, схема задачі, статуси і переходи.
- [MANAGER](MANAGER.md) — Manager: MCP-інструменти, HTTP API, реєстрація в провайдерів.
- [AGENT_LAUNCHER](AGENT_LAUNCHER.md) — запуск агентів, протокол результату worker-а, адаптери Claude, Codex, Cursor, Gemini.
- [AGENT_SESSION_REUSE](AGENT_SESSION_REUSE.md) — одна сесія агента на проєкт і протокол resume.
- [AGENT_TOOL_EVIDENCE](AGENT_TOOL_EVIDENCE.md) — докази, що обов'язкові інструменти справді викликані.
- [PLANE_TASK_PROVIDER](PLANE_TASK_PROVIDER.md) — необов'язковий Plane-провайдер задач.
- [TECHNOLOGY](TECHNOLOGY.md) — стек, збірка, точки входу.
