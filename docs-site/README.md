# Agent Shaker — Documentation Site

Статический сайт с документацией для проекта [Agent Shaker](https://github.com/techbuzzz/agent-shaker), размещаемый на GitHub Pages.

## 🌐 URL сайта

После деплоя сайт будет доступен по адресу:

**https://techbuzzz.github.io/agent-shaker/**

## 🛠 Технологии

- **Next.js 14** с App Router
- **TypeScript** — строгая типизация
- **Tailwind CSS** — стилизация + типографика для документации
- **Статический экспорт** (`output: 'export'`) — полностью статический сайт
- **Remark + remark-gfm** — рендеринг Markdown в HTML
- **Lucide React** — иконки
- **Темная/светлая тема** — переключатель темы

## 📁 Структура проекта

```
docs-site/
├── app/
│   ├── layout.tsx          # Корневой layout с навигацией
│   ├── page.tsx            # Главная страница
│   ├── globals.css         # Глобальные стили + prose для документации
│   ├── screenshots/
│   │   └── page.tsx        # Галерея скриншотов
│   └── docs/
│       ├── page.tsx        # Индекс документации
│       └── [slug]/
│           └── page.tsx    # Просмотр конкретного документа
├── components/
│   ├── Header.tsx          # Навигационная панель (client)
│   ├── Footer.tsx          # Подвал сайта
│   └── ThemeToggle.tsx     # Переключатель темы (client)
├── lib/
│   ├── config.ts            # BASE_PATH / asset() — единый источник basePath
│   ├── docs.ts             # Работа с Markdown документацией
│   └── screenshots.ts      # Работа со скриншотами
├── public/
│   ├── .nojekyll           # Отключает Jekyll на GitHub Pages
│   └── screenshots/        # Изображения (скопированы из ../images/use-cases/)
├── scripts/
│   └── check-export-links.mjs  # Проверка ссылок в готовом out/
├── next.config.js          # Конфиг с basePath и static export
├── tailwind.config.js      # Конфиг Tailwind + типографика
├── postcss.config.js
├── tsconfig.json
└── package.json
```

## 🚀 Локальная разработка

```bash
# Перейти в папку сайта
cd docs-site

# Установить зависимости (один раз)
npm install

# Запустить dev сервер
npm run dev
# Открыть http://localhost:3000/agent-shaker

# Сборка для production
npm run build

# Сборка + проверка всех внутренних ссылок
npm run verify

# Просмотр собранного сайта локально
npx serve out
```

> ⚠️ Dev-сервер тоже работает под `basePath`, поэтому в браузере адрес —
> `http://localhost:3000/agent-shaker` (один сегмент). Двойного `/agent-shaker/agent-shaker/`
> быть не должно: если он появился — см. правило про `asset()` ниже.

## 📦 Деплой на GitHub Pages

Деплой выполняется автоматически через GitHub Actions при push в `main`/`master` ветку.

### Workflow: `.github/workflows/deploy-docs.yml`

Триггеры:
- Push в `main`/`master` с изменениями в `docs-site/`, `docs/` или `images/use-cases/`
- Ручной запуск через вкладку Actions

Этапы:
1. **Build** — установка зависимостей и сборка статического сайта (`npm run build`)
2. **Verify** — проверка внутренних ссылок в `out/` (`npm run check:links`), падает при 404, пропущенном или задвоенном `basePath`
3. **Deploy** — загрузка артефакта и деплой на GitHub Pages

### Включение GitHub Pages

1. Перейти в **Settings → Pages** репозитория
2. В разделе **Source** выбрать **GitHub Actions**
3. После первого деплоя сайт будет доступен по адресу:
   `https://<username>.github.io/agent-shaker/`

## 📝 Добавление документации

Документация автоматически подхватывается из папки `../docs/` (относительно `docs-site/`).

Поддерживаемая структура:
- `docs/*.md` — основные документы (категория «General»)
- `docs/features/*.md` — функции (категория «Features»)
- `docs/releases/*.md` — релизы (категория «Releases»)
- `docs/research/*.md` — исследования (категория «Research»)

### Frontmatter (опционально)

```markdown
---
title: Мой документ
description: Краткое описание для SEO и превью
category: Custom
---

# Заголовок документа
```

## 🖼 Добавление скриншотов

1. Добавить изображения в `../images/use-cases/`
2. Скопировать в `docs-site/public/screenshots/`
3. Добавить метаданные в `lib/screenshots.ts` (название и описание)

## ⚙️ Конфигурация

### next.config.js

```js
{
  output: 'export',           // Статический экспорт
  basePath: '/agent-shaker', // Base path для GitHub Pages
  images: { unoptimized: true }, // Обязательно для static export
  trailingSlash: true,        // Совместимость со статическим хостингом
}
```

### Правило basePath

`basePath` Next.js применяет **сам** — но только к `next/link` и `next/navigation`.

| Контекст | Как писать | Пример |
|---|---|---|
| `<Link href>` / `router.push` | **без** префикса | `href="/docs/quickstart/"` |
| `next/image` `src`, `<img src>`, CSS `url()` | **через `asset()`** | `asset('/screenshots/foo.jpeg')` |

Если писать `href="/agent-shaker/docs/"`, на выходе получится
`/agent-shaker/agent-shaker/docs/` — такой сайт не работает ни на Pages, ни локально.

`asset()` берёт значение из `next.config.js`, поэтому сменить префикс
(например, на кастомный домен) достаточно в одном месте.

### Проверка экспорта

```bash
npm run verify   # build + check:links
```

Скрипт `scripts/check-export-links.mjs` обходит все HTML в `out/` и падает, если:

- ссылка содержит `basePath` дважды;
- внутренняя ссылка не содержит `basePath`;
- `href`/`src` указывает на файл, которого нет в экспорте (404).

Исключения перечислены в `IGNORED_URL_PREFIXES` — это ссылки внутри примеров
кода в документации, а не навигация сайта.

## 📄 Лицензия

Тот же, что и основной проект Agent Shaker.
