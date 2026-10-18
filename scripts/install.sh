#!/bin/sh
# Install or update keenetic-tools on Keenetic / Entware, or remove it with
# everything it created. Run as root:
#   curl -fsSL https://github.com/asiforis/keenetic-tools/releases/latest/download/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- uninstall
# KT_URL overrides where binaries come from (a pinned release: .../releases/download/vX.Y.Z).
set -eu
PATH=/opt/sbin:/opt/bin:/usr/sbin:/usr/bin:/sbin:/bin
export PATH

URL=${KT_URL:-https://github.com/asiforis/keenetic-tools/releases/latest/download}
BIN=/opt/bin/keenetic-tools
CONF=/opt/etc/awg-monitor.conf
CRONTAB=/opt/etc/crontab
CRON_INIT=/opt/etc/init.d/S10cron
OURS='# keenetic-tools: [a-z-]*$'

die() { echo "Ошибка: $*" >&2; exit 1; }

install() {
	A=$(opkg print-architecture | sed -n 's/^arch \([a-z0-9]*\)-k\{0,1\}[0-9].*/\1/p' | sed 's/sf$//' | head -n 1)
	[ -n "$A" ] || die "не удалось определить архитектуру (opkg print-architecture)"
	trap 'rm -f "$BIN.part"' EXIT
	mkdir -p /opt/bin
	curl -fL --progress-bar "$URL/keenetic-tools-$A" -o "$BIN.part"
	sum=$(curl -fsSL "$URL/SHA256SUMS" | awk -v f="keenetic-tools-$A" '$2 == f { print $1 }')
	[ -n "$sum" ] && [ "$(sha256sum "$BIN.part" | cut -d' ' -f1)" = "$sum" ] ||
		die "контрольная сумма keenetic-tools-$A не совпала с SHA256SUMS"
	chmod 755 "$BIN.part"
	v=$("$BIN.part" version) || die "бинарник не запускается на этом роутере ($A)"
	mv -f "$BIN.part" "$BIN"
	echo "Установлен $BIN $v ($A). Дальше — нужные компоненты:"
	echo "  keenetic-tools awg-cron install"
	echo "  keenetic-tools awg-monitor install --log-dir /tmp/mnt/ДИСК/awg-monitor"
}

uninstall() {
	# The binary removes its own components (monitor last: it deletes the binary).
	if [ -x "$BIN" ]; then
		if grep -qs '# keenetic-tools: awg-cron$' "$CRONTAB"; then "$BIN" awg-cron uninstall; fi
		if [ -x "$BIN" ]; then "$BIN" awg-monitor uninstall --purge; fi
	fi
	# What it keeps, or everything if the binary is already gone.
	if grep -qs "$OURS" "$CRONTAB"; then
		sed -i "/$OURS/d" "$CRONTAB"
		if [ -x "$CRON_INIT" ]; then "$CRON_INIT" restart; fi
	fi
	if [ -L /opt/bin/awg-monitor ]; then rm -f /opt/bin/awg-monitor; fi
	if [ "$(head -n 1 "$CONF" 2>/dev/null)" = '# keenetic-tools: awg-monitor' ]; then
		echo "Каталог логов не удалён: $(sed -n 's/^LOG_DIR=//p' "$CONF")"
		rm -f "$CONF"
	fi
	rm -f /opt/etc/crontab.awg-backup.* /opt/etc/crontab.awg-monitor-backup.* \
		/tmp/awg-restart.* /tmp/awg-monitor.* /opt/keenetic-tools.part "$BIN.part"
	echo "keenetic-tools удалён полностью. Пакеты Entware (cron, curl, tcpdump) не тронуты."
}

[ "$(id -u)" = 0 ] || die "запустите под root в shell Entware"
[ ! -e "$BIN" ] || grep -qF '# keenetic-tools: keenetic-tools' "$BIN" ||
	die "$BIN существует и не принадлежит keenetic-tools — не трогаю"
case "${1:-install}" in
install) install ;;
uninstall) uninstall ;;
*) die "использование: sh install.sh [install|uninstall]" ;;
esac
