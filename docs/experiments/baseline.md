# Роутер до установки keenetic-tools и план отката

Исходное состояние рабочего роутера перед первой установкой и как к нему вернуться. Снято
2026-10-25 только чтением, под пользователем `admin` без sudo. Поэтому файлы, закрытые от
чтения не-root пользователю, здесь не описаны (например, `/opt/var/spool/cron`, `sudoers`).

## Состояние «до»

| Что | Значение |
|-----|----------|
| Система | KeenOS, Linux 4.9-ndm-5, `aarch64` (`opkg print-architecture`: `aarch64-3.10`) |
| `/opt` | `/dev/sda1`, ext4, 457 ГБ, занято 2 %. Внутренняя флеш-память: `/` (ubifs, 100 %), `/storage` (ubi) |
| Память | 489 МиБ всего, ~300 МиБ доступно, swap нет; load average ~1.1 в простое |
| Пакеты Entware | 70. Нужные уже стоят: `curl`, `ca-bundle`, `tcpdump`, `sudo`; есть `awg`, `ping`, `ip`, `ndmc` |
| **cron** | **пакета `cron` нет**, `/opt/etc/crontab` и `S10cron` нет, `crond` не запущен. Каталоги `/opt/etc/cron.{1min,5mins,d,hourly,daily,weekly,monthly}` есть, пустые, ничьи по `opkg`. `/opt/var/spool/cron` существует (содержимое закрыто от чтения) |
| keenetic-tools | не установлен: нет `/opt/bin/keenetic-tools`, `/opt/bin/awg-monitor`, `/opt/etc/awg-monitor.conf`, `/tmp/awg-*` |
| Туннель | `amneziawg-go opkgtun0`, init-скрипт `S52awg-opkgtun0` (версия `v2025-12-26 Fin`), конфиг `/opt/etc/amnezia/amneziawg/awg0-opkgtun0.conf` |
| Прочие init.d | `S24xray` (Kvas), `S51dropbear`, `S89amnezia-wg-quick` (`awg0`, сейчас не запущен), `S99adguardhome`, выключенные `K09dnscrypt-proxy2`, `K56dnsmasq` |
| Доступ | SSH `admin` по ключу, `sudo` с паролем; `root` по ключу не пускает |

## Что добавит установка

Существующие файлы установка не перезаписывает, только добавляет новые:

- **пакет `cron`** через `opkg update && opkg install cron`, потому что `S10cron` нет. Вместе с
  ним появятся `/opt/etc/init.d/S10cron`, `/opt/etc/crontab` из пакета и запущенный `crond`.
  `crond` начнёт выполнять и каталоги `/opt/etc/cron.*`; сейчас они пустые;
- файлы keenetic-tools из [docs/artifacts.md](../artifacts.md): бинарник, симлинк, конфиг, строки
  в crontab с бэкапами, каталог логов (по умолчанию `/opt/var/log/awg-monitor`, на `sda1`),
  файлы в `/tmp`;
- `opkg update` обновит списки пакетов в `/opt/var/opkg-lists`. Установленные пакеты он не
  меняет.

Туннель, маршруты и конфигурация Keenetic не меняются. Исключение — `awg-cron`: он
перезапускает `S52awg-opkgtun0` по расписанию.

## Бэкап перед установкой

Выполнить один раз под root (`sudo -i`):

```sh
mkdir -p /opt/root/kt-before && cd /opt/root/kt-before
tar czf etc.tar.gz -C /opt etc
opkg list-installed > opkg.txt
ls /opt/etc/init.d/ > init.d.txt
ls -la /opt/var/spool/cron/ > spool-cron.txt 2>&1
ps w > ps.txt
```

`etc.tar.gz` весит около 1 МБ и содержит ключи AWG. Хранить его на роутере или в надёжном
месте и никогда не коммитить.

## Откат

От мягкого к полному. Каждая команда выполняется под root на роутере и не требует сети и
GitHub.

| Ситуация | Действие |
|----------|----------|
| Мешает захват DNS (нагрузка, память) | `awg-monitor disable capture --purge` |
| Мешает сбор раз в минуту | `awg-monitor disable collect` (данные остаются; вернуть — `enable collect`) |
| Мешает перезапуск туннеля по расписанию | `keenetic-tools awg-cron uninstall` |
| Бинарник не отвечает | `sed -i '/# keenetic-tools: /d' /opt/etc/crontab && /opt/etc/init.d/S10cron restart; kill $(cat /tmp/awg-monitor.dns.pid) 2>/dev/null` |

**Полный откат** к состоянию «до»:

```sh
cd /opt/root/kt-before
awg-monitor disable capture --purge 2>/dev/null
grep -qs '# keenetic-tools: awg-cron$' /opt/etc/crontab && keenetic-tools awg-cron uninstall
[ -x /opt/bin/keenetic-tools ] && keenetic-tools awg-monitor uninstall --purge
# cron до установки не было: убрать пакет, его crontab и init-скрипт
/opt/etc/init.d/S10cron stop
opkg remove cron
rm -f /opt/etc/crontab /opt/etc/crontab.awg-backup.* /opt/etc/crontab.awg-monitor-backup.* \
	/opt/keenetic-tools.part
```

Удалять ли `/opt/etc/crontab`, решаем по таблице «до»: там его не было. `opkg remove`
оставляет изменённые конфиги пакета, поэтому файл удаляется отдельно.

**Проверка после отката:**

```sh
cd /opt/root/kt-before
opkg list-installed | diff opkg.txt -        # пусто
ls /opt/etc/init.d/ | diff init.d.txt -     # пусто
ls /opt/bin/keenetic-tools /opt/bin/awg-monitor /opt/etc/awg-monitor.conf /opt/etc/crontab /tmp/awg-* 2>&1
ps w | grep -E '[c]rond|[t]cpdump'           # пусто
ps w | grep '[a]mneziawg-go'                 # туннель opkgtun0 работает
```

Если всё совпало, `rm -rf /opt/root/kt-before`.

Если что-то пошло совсем не так, весь `/opt/etc` можно вернуть из архива:
`tar xzf /opt/root/kt-before/etc.tar.gz -C /opt`, затем перезагрузить роутер. Это крайняя
мера: архив перезапишет и изменения, сделанные в `/opt/etc` после бэкапа.
