# Site

Публічний сайт `fleet.eggs.gd`. Вихідники — `App/site` у репозиторії App, не окремий репозиторій. Жива сторінка — `sections_v2`.

Випуск на сторінці позначений бейджем Early access. Команда встановлення, `install.sh` і сам Release описані в [release](release.md).

## Зараз

**готово.**

- Hero з анімацією Manager, Execution, Workers, Dashboard, Download.
- Копі hero: manager — це агент, якого людина вже використовує. Fleet дав їй цю роль, а не став нею.
- Секція після hero: одна анімація чату. Повідомлення стає реплікою, з неї вилітає задача в беклог, жовта повертається як Needs Attention, після відповіді летить назад зеленою і сідає в Review. `prefers-reduced-motion` лишає фінальний кадр.
- Бейдж Early access.
- Footer — `fleet.eggs.gd`.
- GitHub Pages збирає `site/` workflow з кореня App.
- Download читає `releases/latest` репозиторія App і без асетів показує Coming soon. Це тимчасовий вигляд: ставлять командою, не кнопкою.

## Далі

Замість кнопки Download — поле з командою `install.sh`. Скрипт і Release описані в [release](release.md). Решти пунктів лендінгу немає. Сторінка вже показує петлю; канал подій у сесію Manager її не блокує.

Dashboard лишається статичним скріншотом: control room, не основний інтерфейс. Кадр на живій сторінці показує реальні проєкти. Заміна на знеособлений кадр — **не зараз**.

## Не зараз

- Окремі сторінки Docs, Support, Changelog, Privacy, Terms, OG/Twitter, canonical, `robots.txt`, sitemap. Юридичні сторінки і дисклеймер товарних знаків потрібні до публічного анонсу; їх ставить [release](release.md).
- Practices і блог. Див. [release](release.md).
