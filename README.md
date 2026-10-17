# keenetic-tools

Перезапуск AmneziaWG-туннеля по расписанию и мониторинг его качества на Keenetic с Entware.
Один статический бинарник; всё выполняется на роутере в shell Entware под root.

## Установка

```sh
opkg update && opkg install curl ca-bundle && A=$(opkg print-architecture | sed -n 's/^arch \([a-z0-9]*\)-k\{0,1\}[0-9].*/\1/p' | sed 's/sf$//' | head -n 1) && echo "Архитектура: $A" && curl -fSL "https://github.com/asiforis/keenetic-tools/releases/latest/download/keenetic-tools-$A" -o /opt/keenetic-tools.part && chmod 755 /opt/keenetic-tools.part && /opt/keenetic-tools.part version
```

Затем установить нужное (бинарник скопирует себя в `/opt/bin/keenetic-tools`):

```sh
/opt/keenetic-tools.part awg-cron install                                         # перезапуск по cron
/opt/keenetic-tools.part awg-monitor install --log-dir /tmp/mnt/HDD/awg-monitor  # мониторинг, логи на внешний диск
rm -f /opt/keenetic-tools.part
```

Обновление — те же шаги; настройки, логи и расписание сохраняются.

## Удаление

```sh
keenetic-tools awg-cron uninstall
awg-monitor uninstall          # логи остаются; --purge удаляет всё
```

Бинарник удаляется вместе с последним компонентом. Пакеты Entware (`cron`, `curl`, `tcpdump`…) не удаляются.

## Команды

| Команда | Что делает |
|---------|------------|
| `keenetic-tools awg-cron install [hourly\|daily]` | перезапуск туннеля каждый час в :01 или раз в сутки в 04:02 |
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
