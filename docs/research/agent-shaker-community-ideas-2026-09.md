# AgentShaker в 2026: куда расти, чтобы попасть в нарратив MCP-сообщества

## Главный тезис

AgentShaker становится релевантным не как «ещё один task tracker», а как **операционный слой для durable multi-agent state** — единственная дыра, которую сейчас затыкают четырьмя-пятью разрозненными инструментами (inbox + file leases + task model + observability + memory). В 2026 году именно вокруг этого слоя формируется нарратив: MCP уже не фича, а инфраструктура [1], A2A ушёл в v1.0 [2], а главный bottleneck сместился с генерации на верификацию [3]. Чтобы зацепиться за этот нарратив, проект должен перестать конкурировать с Jira-клонами и начать конкурировать с operational plane, на который опираются сразу несколько agent runtime.

## Контекст ландшафта

К концу лета 2026 года MCP прошёл точку невозврата: еженедельные загрузки npm-пакета `@modelcontextprotocol/sdk` выросли с ~35,5 млн (июнь) до ~52,6 млн (август), что больше, чем у OpenAI и Anthropic SDK по отдельности [1]. Даже HN-обсуждение «MCP is dead?» с 399 очками и 410 комментариями не остановило рост — экосистема превратилась в load-bearing инфраструктуру, и любое решение про координацию агентов уже вынуждено считаться с MCP-примитивами. Параллельно вендоры начали строить multi-agent «из коробки»: VS Code 1.109 (январь 2026) выкатил Agent HQ с делегированием между Claude/Codex/Copilot, Claude Code Agent Teams уехали в mail-box + git-locking (февраль 2026), Codex multi-agent v2 добавил path-based адресацию (март 2026) [4][5]. Это уже не лабораторные конструкции — команды в проде гоняют трёх-четырёх агентов одновременно, и именно на стыке между ними возникает тот самый «operational layer», которого пока нет как готового продукта.

A2A тем временем дорос до v1.0 (март 2026), v1.0.1 (май), v1.1.x (сентябрь) — то есть спецификация стабилизировалась быстрее, чем кто-либо ожидал, и появились SDK на JS/Python/Go [2]. Это означает, что «агент-как-MCP-сервер» перестаёт быть демкой: можно проектировать продукт, в котором агенты становятся first-class participants. AgentShaker уже реализовал A2A — это редкое преимущество, которое сейчас почти никто из конкурентов не закрыл.

## Боли, которые сообщество называет вслух

Прежде чем выбирать фичи, полезно сверить, что именно HN, Reddit и GitHub Issues в 2025–2026 называют настоящей болью, а не AI-хайпом. Картина получается довольно конкретная.

**Деградация памяти.** Эталонный бенчмарк Mem0 OSS на 50K сессий показывает падение accuracy с 91,6% до 49% за 30 дней, 38% записей становятся stale [6]. Причины известны: pure-vector хранилища не различают «обновили» и «добавили новое» — Mem0 хранит обе версии, Letta OSS не делает contradiction check, LangMem делает blind-overwrite через `put`, сбрасывая `created_at` [6]. Без явного contradiction detection и decay любая память превращается в свалку к 5K записям.

**Verification bottleneck.** В продакшене 41–86,7% multi-agent пайплайнов ломаются без формальной оркестрации [7]. Из этих поломок 41% — specification design, 36,9% — inter-agent misalignment (потерянный контекст, плохой handoff, формат), 21,3% — verification failure [7]. Это согласуется с трендом 2026 «verification is the bottleneck»: появились отдельные категории инструментов — TDAD (Test-Driven Agent Definition), ASSERT, Harness Agent DLC [3]. Запрос «лучше спеки, а не более умные модели» — устойчивый, не хайповый.

**Handoff failures.** «Большинство провалов — это провалы handoff» — повторяющаяся тема Reddit. Сравнение структурированных handoff-документов против свободного prose даёт 60% reduction в cross-agent errors при росте токенов всего на 20% [8]. Это прямой сигнал: агенту нужно передавать не «весь диалог», а typed contract.

**Memory poisoning и gossip.** 21% поломок — отравление памяти, и отдельная категория жалоб — «агенты пишут что-то за моей спиной» (humans не видят, что агенты туда записали) [7]. Без auditable trail это превращается в доверительную яму.

**Setup ceremony.** В энтерпрайзе для production-ready AI-coding agent требуется 7 контролей (SSO, SIEM, secret scanning, PR policy, license governance, sandboxing, incident response), on-boarding сократился с недель до часов только там, где есть выделенный agent owner — а это 56% scaling-компаний [9]. Малые команды по-прежнему отваливаются на этапе «как это вообще поднять».

## Конкурентная карта

В нише «task/project tracker for AI agents» за последние 10 месяцев на Show HN вышло минимум семь отдельных запусков — от Buildable (июнь 2025) до BigBlueBam (апрель 2026), плюс несколько вне HN. Это формирующаяся, но ещё не устоявшаяся категория. Сводная таблица по проектам, которые имеют значение для AgentShaker:

| Проект | Ядро | Стек | Протоколы | Сильная сторона | Слабое место |
|---|---|---|---|---|---|
| **AgentShaker** | Projects → Agents → Tasks → Contexts | Go + PostgreSQL + Vue | REST + MCP + WebSocket + A2A + OTel | Единственный production-combo всех пяти протоколов сразу; durable store | Нет CLI, нет git-as-coordination, нет memory layer, нет верификации |
| **Beads** (steveyegge) | Git-as-DB задачи с dependency graph и atomic claims | Go + Dolt/SQLite + JSONL | CLI + (опц.) MCP | Чистый narrative, огромный launch-трафик, «50 First Dates» framing | CLI-only, нет A2A, нет веб-дашборда, нет мультиагент-семантики [10] |
| **MCP Agent Mail** (Dicklesworthstone) | Inbox + file leases для агентов | Python + FastMCP + SQLite | MCP | File leases, identity model, async inbox | Нет task model, нет A2A, нет durable multi-task store [11] |
| **Agent Hub MCP** (gilbarbara) | Универсальный coordination hub | TypeScript | MCP + message bus | Show HN, npm-пакет, message bus | Нет durable task state, нет handoff schema [12] |
| **Mycelium MCP** | Memory + tasks + messaging + oversight | — | MCP | Самая широкая поверхность в одном MCP-сервере | Качество отдельных модулей неравномерно |
| **mcp-agent** (LastMile AI) | Python framework для построения агентов | Python | MCP + Temporal | Workflow patterns, Temporal support | Это framework, не tracker; mcp-c облако в open beta [13] |
| **BigBlueBam** | Work OS с agents как first-class | MIT, open source | MCP + REST | Самый амбициозный scope, недавний запуск | Ранний, маленькая база, нет A2A |
| **Cueit / Orchestro / VibeCoCo / Hive Memory** | Нишевые варианты: лёгкий Kanban, Trello-for-Code-Code, генератор per-project серверов, cross-project memory | Разный | В основном MCP | Каждый закрывает одну нишу | Ни один не предлагает фундамента |
| **Hub (Slate)** | Trust/obligation/attestation framework | — | MCP | Самый глубокий governance-стек | Нишевый, мало кто знает |
| **Mastra 101** | MCP-as-curriculum — учебный курс в форме MCP-сервера | — | MCP | HN 213 pts — доказал, что MCP-as-content работает | Не продукт, не конкурент по функциям [14] |

Из таблицы видно: **ни один проект не закрывает одновременно MCP + REST + WebSocket + A2A + durable PostgreSQL store в одной коробке**. Большинство выбирает одно-два и компенсирует остальное соседними инструментами. Это и есть окно, в которое может попасть AgentShaker — если перестанет прятаться за словом «tracker».

## Сильные стороны и дыры AgentShaker

У проекта уже есть редкие capability-combo: A2A-имплементация (контрактная стабилизация), PostgreSQL как operational class против SQLite/JSONL, OpenTelemetry-интеграция, WebSocket для live updates, нативная интеграция с VS 2026 через `.mcp.json` и скрипты копирования [15]. Это означает, что любой разговор об «operational layer для multi-agent state» AgentShaker может вести с позиции, а не догонять.

Дыры, которые нельзя игнорировать: нет CLI-инструмента с `bd ready`-паритетом, нет git-as-coordination (Beads и MCP Agent Mail выигрывают здесь по умолчанию), нет memory layer с contradiction detection и decay, нет verification harness, нет inbox/threading (Agent Hub опережает), нет Skills-marketplace экспорта (отсутствие соответствия agentskills.io [16]), нет публичного трейла, который показывает людям, что агенты записали в contexts. Наконец, нет видимого community-канала — ни Show HN в недавней памяти, ни активного GitHub Discussions, ни Discord, что в этой нише критично, потому что доверие строится через публичные демо.

## Идеи для роста relevance

Ниже — девять идей, отобранных из двенадцати рассмотренных. Идеи сгруппированы по временному горизонту и типу риска. Внутри каждой группы указан закрываемый pain, конкурентный фон и что делать первым шагом.

### Фундамент (4–8 недель, устойчивое преимущество)

**Memory layer с contradiction detection и decay.** Самая болезненная точка сообщества — деградация памяти, и единственное структурное отличие AgentShaker от конкурентов — уже-structured PostgreSQL store. Можно добавить таблицу `context_relations` с типами `supersedes`, `contradicts`, `expires_at`, плюс background-job, который раз в час ищет противоречия и предлагает агенту/человеку резолюцию. Decay можно считать через экспоненту от `last_used_at`. Mem0, Letta и LangMem все здесь облажались по-разному [6], и это дыра, которую PostgreSQL закрывает естественно. Стартовый шаг: спроектировать схему `contexts + context_relations`, сделать миграцию, добавить REST/MCP-метод `contexts.audit()` и UI-виджет «stale memories» в дашборде. Метрика успеха: доля stale contexts, обнаруженных автоматически, и количество merge-операций за неделю.

**Verification harness как first-class task state.** Текущая модель «task → done» бинарна и не отличает «агент написал done» от «агент реально проверил». Можно ввести состояния `claimed → executing → verifying → done` с обязательным attachment evidence (test output, screenshot, log hash) перед переходом в `done`. Это ложится прямо на A2A handoff и превращает AgentShaker в единственный MCP-PM, который закрывает verification bottleneck. Стартовый шаг: расширить `task.status` enum, добавить `task.attach_evidence()` в MCP, описать политику в `docs/VERIFICATION.md`. Метрика: % done-тасков с прикреплённым evidence и количество rollback-операций.

**Typed handoff contracts поверх A2A.** A2A v1.0 даёт envelope, но не схему payload. AgentShaker может опубликовать канонические JSON Schema для типовых handoff (handoff.developer_to_reviewer, handoff.pm_to_engineer, handoff.escalation), включить их в `internal/a2a/schemas/`, сделать валидацию по умолчанию и опубликовать примеры. Это даёт «20% больше токенов, 60% меньше ошибок» [8] из коробки, плюс делает AgentShaker reference implementation для A2A в этой нише. Стартовый шаг: выбрать 3 handoff-сценария из текущего InvoiceAI demo, описать схемы, добавить валидацию в `internal/a2a/server`.

### Быстрый PR (1–3 недели, видимость и трафик)

**CLI-инструмент с минимальной поверхностью.** Даже тонкий `agent-shaker` CLI (`ready`, `claim`, `handoff`, `done --evidence path`) превращает проект из «ещё один дашборд» в developer-first tool. Beads выстрелил именно потому, что `bd ready` оказался в shell-истории каждого пользователя. Стартовый шаг: новый `cmd/agent-shaker-cli`, использовать Cobra, выкатить `homebrew tap` и `scoop bucket`. Метрика: количество звёзд за месяц и упоминаний CLI в issue-трекере.

**Регистрация в MCP-директориях.** Это бесплатный трафик, который не используется. Glama — ~505K посетителей/мес, PulseMCP — 277K, Smithery — 446K, Official Registry — 54K, mcp.so — 238K, MCP Market — 1,4M (правда, низкий DR) [17]. Один weekend-task: подготовить `server.json` со schema, screenshots, capabilities, оформить README в формате каталога, засабмитить в первые пять. Стартовый шаг: следовать `modelcontextprotocol/registry` guidelines, завести PR в Glama + Official Registry. Метрика: количество install-through из каждой директории.

**Show HN: «coordination plane for AI agents, not another Jira clone».** Это готовый narrative — Victor его уже сформулировал [18]. Show HN Mastra 101 набрал 213 очков именно на MCP-as-curriculum framing [14]; для AgentShaker естественный кадр — «operational layer alongside Jira/Slack, not a replacement». Подготовка: 90-секундное видео-демо (агент входит в проект, берёт задачу, делает handoff, завершает с evidence), короткий README, страница с графиками (latency, throughput, cost), и честный раздел «what this is not». Стартовый шаг: записать screencast, написать пост, выкатить в среду утром EST. Метрика: HN-points ≥ 100, GitHub stars delta за неделю ≥ +500.

**Native SKILL.md export и соответствие agentskills.io.** Это самая низко висящая фича в 2026. Anthropic Skills стали открытым стандартом [16], SkillsMP индексирует 1,2M+ skill-бандлов [19], marketplace primitive закрепился в Claude Code 2.1+ (январь 2026). Можно одной командой (`agent-shaker export skill <workflow-id>`) превращать workflow AgentShaker в SKILL.md бандл с YAML-frontmatter, прогрессивной загрузкой и `allowed-tools`. Это даёт проекту канал дистрибуции в экосистему Claude Code / Codex, где сидит реальная база пользователей. Стартовый шаг: модуль `internal/skills/export`, шаблон SKILL.md, CLI-команда `export skill`. Метрика: количество скачиваний сгенерированных skills, упоминания в awesome-claude списках.

### Эксперименты (1–6 месяцев, ставка на новую категорию)

**AgentShaker как реестр skills/workflows, а не задач.** Если пойти на шаг дальше, чем export, можно превратить проект в registry процедурных знаний для агентов — то, чего в экосистеме пока нет. Workflows в AgentShaker уже markdown, уже структурированы (projects → agents → tasks → contexts), значит превращение в publishable skills — это в первую очередь schema-mapping, а не новый продукт. Стартовый шаг: research-spike на 2 недели — посмотреть, какие из существующих workflows естественно ложатся на SKILL.md, поговорить с 5 потенциальными пользователями. Метрика: retention-pubrate (опубликованные skill / зарегистрированные workflow) ≥ 30%.

**Git-as-coordination primitive (опциональный режим).** Это самая амбициозная ставка: позволить агентам брать задачи в Git worktree-режиме с file leases, как MCP Agent Mail [11] и Claude Code Agent Teams [4]. Здесь нужна осторожность — это другой продукт, и попытка закрыть всё сразу убьёт фокус. Стартовый шаг: feature flag за MCP-методом `task.claim_git_worktree`, документировать как experimental, не продвигать в marketing до получения реальных пользователей. Метрика: opt-in rate и качество обратной связи в первые 30 дней.

## Top-3 на ближайший квартал

**Приоритет 1 — Memory layer с contradiction detection.** Четыре-восемь недель. Закрывает pain №1 сообщества, использует уникальную PostgreSQL-природу проекта, не имеет конкурента с качественной имплементацией. Риск — scope creep в «AI memory product», поэтому строго держаться структурной части (schema + audit) и не делать LLM-выводов «что правильно».

**Приоритет 2 — Параллельный bundle: CLI + MCP-директории + Show HN + SKILL.md export.** Одна-две недели на каждую часть, всё в одном релизе. Этот bundle — единый narrative: «AgentShaker — это operational layer, доступный из shell, из MCP, из Claude Code Skills и из web-дашборда». Show HN даёт visibility, директории дают install-throughput, CLI даёт habit-forming touchpoint, SKILL.md export даёт дистрибуцию в экосистему. Риск — половина мер thin, поэтому либо делать всё, либо только CLI + Show HN.

**Приоритет 3 — Verification harness.** Шесть-десять недель, после данных от приоритетов 1 и 2. Закрывает pain №2 (verification bottleneck), естественно ложится на A2A handoff, даёт второе устойчивое преимущество после memory layer. Риск — слишком opinionated UX; митигация — стартовать с минимального набора evidence-типов (test-output, git-commit, screenshot) и расширять по запросам.

## Каналы дистрибуции

Show HN — единственный канал с непропорционально высоким leverage: Mastra 101 получил 213 pts на MCP-as-curriculum framing [14], mcp-c получил cloud-platform visibility от того, что LastMile уже имел HN-историю [13]. Reddit r/AI_Agents и r/ClaudeAI — вторичный канал, естественный hook — memory degradation и verification. GitHub Discussions и Discord — обязательный table stakes, но не launch-канал. X/Twitter — отражатель HN, не генератор. Из директорий в первую очередь Official Registry + Glama + Smithery + PulseMCP [17]. dev.to и личный блог — длинный пост «memory degradation vs durable state», который объясняет позицию. Ключевое правило: одна история, рассказанная в разных форматах для разных аудиторий, а не десять отдельных историй.

## Анти-паттерны

«Добавьте AI-фичи» — насыщает, не дифференцирует. «Запишитесь в OpenAI marketplace» — такого нет, либо не решает relevance. «Сделайте SaaS-версию» — меняет продукт, а не задачу. Гнаться за каждым Show HN форматом — лучше одна история, рассказанная хорошо. Переписывать на TypeScript/Rust — Go сейчас ваше edge (perf + простота). Заводить Discord до traction — кладбище. Конкурировать по фичам Jira (board, sprint) — уже проигрыш; позиционироваться надо как дополняющий слой, а не замена [18]. Наконец, строить собственный vector store для memory — PostgreSQL с structured relations закрывает задачу, и это именно то, чего нет у конкурентов.

## Источники

[1] https://www.builder.io/blog/state-of-ai-2026 и https://www.towardsai.net/p/state-of-ai-in-2026 — MCP SDK npm download trends, 35,5M (Jun 2026) → 52,6M (Aug 2026).  
[2] https://github.com/a2aproject/A2A — A2A v1.0 spec, v1.0.1, v1.1.x timeline, JS/Python/Go SDKs.  
[3] https://www.developersdigest.org/ — TDAD/ASSERT/Harness Agent DLC, «verification is the bottleneck» 2026.  
[4] https://code.visualstudio.com/blogs/2026/01/22/agentHQ — VS Code 1.109 Agent HQ (Jan 2026).  
[5] https://docs.claude.com/en/docs/claude-code/agent-teams — Claude Code Agent Teams mailbox + git-locking (Feb 2026); https://github.blog/ — Codex multi-agent v2 path-based (Mar 2026).  
[6] https://www.ranksquire.com/blog/mem0-vs-langmem-vs-letta-comparison/ — Mem0 accuracy decay 91,6→49% over 30 days, 38% staleness, contradiction-handling comparison.  
[7] https://www.researchgate.net/publication/AI-Multi-Agent-Failure-Modes — 41–86,7% failure rate; spec 41%, handoff 36,9%, verification 21,3%.  
[8] https://www.reddit.com/r/AI_Agents/comments/structured-handoff-vs-prose — Reddit structured-handoff 60% fewer errors vs 20% token increase.  
[9] https://www.anthropic.com/enterprise-agent-readiness-2026 — 7 enterprise controls, agent-owner role in 56% scaling companies.  
[10] https://github.com/steveyegge/beads — Beads: Go + Dolt/SQLite, git-as-DB, 6-day build, atomic claims.  
[11] https://github.com/Dicklesworthstone/mcp-agent-mail — Inbox + file leases, FastMCP, Python.  
[12] https://github.com/gilbarbara/agent-hub-mcp — Agent Hub MCP universal coordination, TypeScript.  
[13] https://github.com/lastmile-ai/mcp-agent и https://news.ycombinator.com/item?id=45693834 — mcp-agent framework + mcp-c cloud.  
[14] https://startups.center/zh/w/mastra-2025-course-as-mcp-server-bet — Mastra 101 Show HN 213 pts, MCP-as-curriculum framing.  
[15] https://github.com/techbuzzz/agent-shaker/blob/main/README.md и /VS2026_DELIVERY_SUMMARY.md — AgentShaker capabilities.  
[16] https://github.com/anthropics/skills и https://agentskills.io — SKILL.md open standard, marketplace primitive.  
[17] https://servicegraph.co/blog/best-mcp-directories-2026 и https://www.innvesti.com/reports/best-mcp-server-directories-2026/ — MCP directory traffic (Glama 505K, Smithery 446K, mcp.so 238K, PulseMCP 277K, Official 54K, MCP Market 1,4M).  
[18] https://www.linkedin.com/in/victor-buzin/ — Victor Buzin: «coordination plane that AI agents can interact with via MCP», «sit alongside Jira, Slack, and your existing doc stack».  
[19] https://www.totalum.app/blog/claude-skills-marketplace-totalum и https://skillsmp.com/ — SkillsMP 1,2M+ skills, marketplace growth.
