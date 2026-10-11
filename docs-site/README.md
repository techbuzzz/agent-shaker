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
│   ├── docs.ts             # Работа с Markdown документацией
│   └── screenshots.ts      # Работа со скриншотами
├── public/
│   ├── .nojekyll           # Отключает Jekyll на GitHub Pages
│   └── screenshots/        # Изображения (скопированы из ../images/use-cases/)
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

# Просмотр собранного сайта локально
npx serve out
```

## 📦 Деплой на GitHub Pages

Деплой выполняется автоматически через GitHub Actions при push в `main`/`master` ветку.

### Workflow: `.github/workflows/deploy-docs.yml`

Триггеры:
- Push в `main`/`master` с изменениями в `docs-site/`, `docs/` или `images/use-cases/`
- Ручной запуск через вкладку Actions

Этапы:
1. **Build** — установка зависимостей и сборка статического сайта (`npm run build`)
2. **Deploy** — загрузка артефакта и деплой на GitHub Pages

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

## 📄 Лицензия

Тот же, что и основной проект Agent Shaker.
