# Что keenetic-tools создаёт на роутере

Список всего, что появляется на роутере, — чтобы при необходимости удалить вручную.
Обычно хватает `install.sh uninstall` (см. README) — он удаляет всё из списка ниже, кроме пакетов Entware.

## Общее

| Путь | Что это | Кто создаёт |
|------|---------|-------------|
| `/opt/bin/keenetic-tools.part` | недокачанный бинарник, если `install.sh` прервали | `install.sh`; удаляет он же |
| `/opt/bin/keenetic-tools` | установленный бинарник | `install` любого компонента |
| строки в `/opt/etc/crontab` с `# keenetic-tools: ...` | задания cron | `install` |
| пакет `cron` (`/opt/etc/init.d/S10cron`) | ставится через `opkg`, если его нет | `install` |

## awg-cron

| Путь | Что это |
|------|---------|
| `/opt/etc/crontab`: строка `... awg-cron run # keenetic-tools: awg-cron` | расписание перезапуска |
| `/opt/etc/crontab.awg-backup.XXXXXX` | копии crontab перед каждым изменением (не удаляются никогда) |
| `/tmp/awg-restart.log` | лог последнего перезапуска (RAM) |
| `/tmp/awg-restart.lock` | блокировка от параллельного запуска (RAM) |

## awg-monitor

| Путь | Что это |
|------|---------|
| `/opt/bin/awg-monitor` | симлинк на `/opt/bin/keenetic-tools` |
| `/opt/etc/awg-monitor.conf` | настройки; первая строка `# keenetic-tools: awg-monitor` |
| `/opt/etc/crontab`: строка `... awg-monitor collect ... # keenetic-tools: awg-monitor` | ежеминутный сбор |
| `/opt/etc/crontab.awg-monitor-backup.XXXXXX` | копии crontab (удаляются с `uninstall --purge`) |
| `/tmp/awg-monitor.*` | RAM: `.lock`, `.nodir`, захват DNS (`.dns.log*`, `.dns.pid`, `.dnsmap`, `.ctseen`, `.includes`) |
| процесс `tcpdump` | только при включённом `capture`; PID в `/tmp/awg-monitor.dns.pid` |
| syslog, тег `awg-monitor` | события; живут до очистки журнала / перезагрузки |

**Каталог логов** — значение `LOG_DIR` в конфиге (по умолчанию `/opt/var/log/awg-monitor`):

| Путь | Что это |
|------|---------|
| `.awg-monitor` | маркер каталога: без него `collect` ничего не пишет |
| `samples/ДАТА.log` | ежеминутные замеры |
| `domains/ДАТА.log`, `domains.state` | проверки доменов из `DOMAINS_WATCH` |
| `missed/ДАТА.log`, `missed.cache` | соединения мимо туннеля (IP устройств и домены) |
| `events.log`, `events.log.1` | события |
| `state` | состояние между запусками |
| `dumps/ДАТА_ВРЕМЯ-ПРИЧИНА.txt` | дампы при переходе в `DOWN` |
| `snapshots/ДАТА_ВРЕМЯ[-метка].txt` | снимки конфигурации (без ключей) |
| `bundles/awg-monitor-ДАТА_ВРЕМЯ.tar.gz` | архивы `bundle` |
| `*.tmp.PID` | недописанные файлы после сбоя |

При смене `--log-dir` старый каталог остаётся как был.

## Удалить вручную

```sh
# остановить
sed -i '/# keenetic-tools: /d' /opt/etc/crontab && /opt/etc/init.d/S10cron restart
kill $(cat /tmp/awg-monitor.dns.pid) 2>/dev/null
# каталог логов — посмотреть до удаления конфига
grep '^LOG_DIR=' /opt/etc/awg-monitor.conf
rm -rf 'ЗНАЧЕНИЕ_LOG_DIR'
# файлы
rm -f /opt/bin/awg-monitor /opt/bin/keenetic-tools /opt/bin/keenetic-tools.part /opt/etc/awg-monitor.conf
rm -f /opt/etc/crontab.awg-backup.* /opt/etc/crontab.awg-monitor-backup.*
rm -f /tmp/awg-restart.* /tmp/awg-monitor.*
# пакеты, если не нужны другим программам
opkg remove tcpdump cron curl ca-bundle
```
