# keenetic-tools

Перезапуск AmneziaWG-туннеля по расписанию и мониторинг его качества на Keenetic с Entware.
Один статический бинарник; всё выполняется на роутере в shell Entware под root.

## Установка

```sh
opkg update && opkg install curl ca-bundle
curl -fsSL https://github.com/asiforis/keenetic-tools/releases/latest/download/install.sh | sh
```

`install.sh` определяет архитектуру роутера, скачивает бинарник последнего релиза, сверяет его
с `SHA256SUMS` и ставит в `/opt/bin/keenetic-tools`. Затем — нужные компоненты:

```sh
keenetic-tools awg-cron install                                  # перезапуск туннеля по cron
keenetic-tools awg-monitor install --log-dir /tmp/mnt/HDD/awg-monitor   # мониторинг, логи на внешний диск
```

Обновление — снова `install.sh`; настройки, логи и расписание сохраняются.

## Удаление

```sh
curl -fsSL https://github.com/asiforis/keenetic-tools/releases/latest/download/install.sh | sh -s -- uninstall
```

Удаляет всё, что создал keenetic-tools: задания cron, бинарник, конфиг, логи, резервные копии
crontab, файлы в `/tmp`. Чужие задания cron и пакеты Entware (`cron`, `curl`, `tcpdump`…) не трогает.
По одному компоненту: `keenetic-tools awg-cron uninstall`, `awg-monitor uninstall [--purge]`.
Полный список файлов и ручное удаление — [docs/artifacts.md](docs/artifacts.md).

## Команды

| Команда | Что делает |
|---------|------------|
| `keenetic-tools awg-cron install [10min\|hourly\|daily]` | перезапуск туннеля каждые 10 минут (:01, :11, … :51), каждый час в :01 или раз в сутки в 04:02 |
| `keenetic-tools awg-cron run` | перезапустить туннель сейчас (лог — `/tmp/awg-restart.log`) |
| `keenetic-tools version` | версия |
| `awg-monitor status` | что установлено и запущено, сколько данных; состояние туннеля |
| `awg-monitor check` | замер прямо сейчас, без записи |
| `awg-monitor report [24h\|7d]` | время по состояниям `OK`/`DEGRADED`/`DOWN`, сбои с причинами, RTT, память |
| `awg-monitor enable\|disable collect` | включить/выключить ежеминутный сбор |
| `awg-monitor enable\|disable capture [--purge]` | поиск доменов мимо туннеля (нужен `tcpdump`); `--purge` удаляет собранное |
| `awg-monitor missed [24h\|7d]` | домены и соединения, ушедшие мимо туннеля без ответа, с готовыми строками `include` |
| `awg-monitor check-domain ДОМЕН...` | открыть домен напрямую и через туннель, показать этап обрыва (`dns`/`tcp`/`tls`/`http`/`cert`) |
| `awg-monitor domains [24h\|7d]` | сводка постоянных проверок доменов из `DOMAINS_WATCH` |
| `awg-monitor route-check ДОМЕН` | почему домен идёт или не идёт через туннель |
| `awg-monitor snapshot [метка]` / `snapshots` | снять конфигурацию / список снимков |
| `awg-monitor diff [A] [B]` | разница снимков (по умолчанию два последних) |
| `awg-monitor bundle [дней]` | архив логов, снимков и дампов для разбора (без ключей) |

`awg-monitor` — то же, что `keenetic-tools awg-monitor`. Настройки — `/opt/etc/awg-monitor.conf`.

## SSH-пользователь с sudo

Отдельный скрипт: создаёт пользователя с ключами из `/opt/root/.ssh/authorized_keys` и правом `sudo`.

```sh
opkg update && opkg install curl ca-bundle && curl -fSL https://raw.githubusercontent.com/asiforis/keenetic-tools/main/scripts/add-ssh-user.sh -o /opt/add-ssh-user.sh.part && mv /opt/add-ssh-user.sh.part /opt/add-ssh-user.sh && sh /opt/add-ssh-user.sh admin
```

## Разработка

Сборка — кросс-компиляция на ПК (Linux, macOS, WSL; нужны Go и [Task](https://taskfile.dev)),
на роутере ничего не собирается. `task` — список задач, `task check` — всё, что проверяет CI,
`task deploy ROUTER=root@192.168.1.1 ARGS='awg-monitor check'` — собрать под роутер, запустить и удалить,
`task verify TAG=vX.Y.Z` — пересобрать релиз и сверить с его `SHA256SUMS`.
