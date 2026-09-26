# Manager: skills, код і MCP

Належить App. Роадмап: [app](../roadmaps/app.md).

**немає.** Цього в репозиторії ще немає. Нижче — що робити, не поточний стан.

Принцип: код робить те, що має одну правильну відповідь (ID, слаги, статуси, дедуп, валідація). MCP віддає компактні факти і приймає атомарні записи. Skill тримає лише судження, яке не звести до правил: коли питати, як різати задачу, що вважати готовим.

Захист локального API (Host, Origin, токен, не-JSON) уже стоїть. Нові писальні інструменти нижче йдуть на той самий захищений порт.

## Де зараз горять токени

Manager читає `_docs/MANAGER.md`, `TASK_LIFECYCLE.md`, `OPERATING_MODEL.md` і `Fleet/ROUTING.md` у data root. Разом близько 46 КБ, приблизно 12k токенів на холодний старт. `Work/INDEX.md` — ще близько 36 КБ. Якщо він зазирає в `_registry/repositories.json`, це сотні кілобайт. `manager_vocabulary` уже дає компактну проєкцію, але інструкції все одно кажуть читати файли.

Більшість правил у `MANAGER.md` уже виконує код. Manager читає їх як інструкції, а не як контракт: ref-и, слаги, шаблон картки, priority за замовчуванням, `needs_rework` перед `todo`.

## Skills

Кожен skill — самодостатня інструкція до приблизно 80 рядків: рішення, приклади, точні виклики інструментів. Без посилань на документи. Не більше цих семи.

Формат: рідні skills провайдера (Claude Code `.claude/skills`, Codex skills) плюс запасний варіант через MCP `prompts`. Один джерельний текст живе в App і версіонується з ним. Provisioning уже пише MCP-конфіги; файли skills пишуться тим самим кроком.

1. **intake** — сирий ввід (голос, чат) стає одним із п'яти результатів: задача, Inbox, оновлення проєкту, архів, нічого. Поріг: коли це вже задача, а коли питати. Сирий текст завжди зберігається. Закінчується одним викликом MCP.
2. **shape-task** — з наміру зробити ціль, критерії приймання, тип, пріоритет. Коли створювати research-задачу, яка блокує імплементацію, а не вигадувати специфікацію.
3. **resolve-project** — як читати `manager_resolve_project`: `confident`, `ambiguous`, `none`. Коли ставити `backlog` і `unassigned` і питати людину. Коли зберігати аліас.
4. **route** — коли `manager_route` віддав низьку впевненість або конфлікт: до кого і чому, що записати в handoff.
5. **triage-attention** — реакція на `needs_attention` і `needs_review`: стисло переказати питання людині, прийняти відповідь, передати її worker-у, не розростатись.
6. **review** — порівняти результат з критеріями приймання: accept чи rework, як сформулювати review-коментар. Ownership звільняється лише після Accepted.
7. **briefing** — голосовий підсумок дошки: що біжить, що заблоковано, що чекає рішення. Кілька речень.

## Читання

Компактні факти замість файлів.

- `manager_board(view)` — `needs_attention`, `blocked`, `in_review`, черги по репозиторіях, хто тримає ownership. Замінює читання `INDEX.md`.
- `manager_task(ref)` — картка, залежності, блокери, стан сесії, хвіст activity log.
- `manager_workers()` — які агенти доступні, чиї ownership, що вільне. Discovery агентів уже є в коді.
- `manager_events(since)` — poll подій `task.*` за курсором. У сесію Manager їх ніхто не пушить.

## Рішення з відповідальністю

Правило в коді. Manager лише погоджує.

- `manager_resolve_project(phrase)` — ранжовані кандидати, оцінка і вердикт `confident`, `ambiguous` або `none`. Кодифікує правила резолву проєкту.
- `manager_similar(text)` — дублікати і суміжні задачі перед створенням.
- `manager_route(draft)` — виконавець за пріоритетом: явна вказівка, override проєкту, категорія з маршрутизації, доступність worker-а. `ROUTING.md` стає конфігом, не прозою.
- `manager_validate(draft)` — dry-run: схема, Definition of Done, відсутні репозиторії, цикли `depends_on`. Повертає список виправлень.

## Запис

Одна атомарна дія замість послідовності.

- Розширити `manager_command`. У `Intent` зараз є однина `repository`, і немає `depends_on`, `repositories`, `acceptance_criteria`, `source_inbox`. Без них Manager править картку руками.
- `manager_inbox(capture | promote | list)` — INBOX-ref-и, промоція в CORE-задачу з `source_inbox` і `promoted_to`. Зараз цього немає.
- `manager_project(alias | note | decision)` — ідемпотентно, без дублів, із записом в activity log. Замінює ручне редагування `PROJECT.md`.
- `manager_answer(ref, text)` — коментар, перехід статусу і продовження worker-а однією дією.
- `manager_review(ref, verdict, comment)` — accept веде в done і звільняє ownership; rework веде в `needs_rework` з коментарем у стандартній секції.
- `manager_decompose(parent, subtasks)` — research і implementation з `depends_on` атомарно, у правильному порядку.

Правила полів (default priority, допустимі статуси) живуть у `description` полів JSON Schema. Інструмент навчає сам, окремий документ для цього не потрібен.

Помилки — `fail-with-fix`: типізована помилка з підказкою («project не знайдено, найближчі: …»), щоб модель не ганяла петлі.

## Що забрати з документів data root

- `MANAGER.md` розкласти: правила ID, слагів, refs, priority і шаблону картки стають кодом і схемою; судження йдуть у skills; решта зникає.
- `TASK_LIFECYCLE.md` і `OPERATING_MODEL.md` лишити короткими довідками для людини, не інструкціями Manager.
- В `AGENTS.md` data root для Manager лишити три речі: ти Manager, працюй тільки через MCP, до файлів не торкайся.

## Порядок

1. `manager_board` і `manager_task` плюс skill `intake`. Найбільша економія токенів і найнижчий ризик.
2. Розширити `Intent` (`depends_on`, `repositories`, `acceptance_criteria`) і додати `manager_validate`.
3. `manager_resolve_project`, `manager_similar`, `manager_route`. Правила маршруту — у конфігу.
4. `manager_inbox`, `manager_project`.
5. `manager_answer`, `manager_review`, `manager_decompose` разом зі skills `triage-attention` і `review`.
