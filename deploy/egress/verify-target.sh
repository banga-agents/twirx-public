#!/usr/bin/env bash
set -euo pipefail

unit_path=${1:-/etc/systemd/system/twirx-egress-worker@.service}

if [[ ! -f "${unit_path}" ]]; then
  echo "FAIL unit_missing=${unit_path}"
  exit 1
fi

systemd-analyze verify "${unit_path}"

required_directives=(
  'User=twirx-egress'
  'NoNewPrivileges=true'
  'ProtectSystem=strict'
  'SocketBindDeny=any'
  'IPAddressDeny=127.0.0.0/8'
  'IPAddressDeny=169.254.0.0/16'
  'IPAddressDeny=::1/128'
  'IPAddressDeny=fc00::/7'
  'UMask=0077'
  'MemoryMax=256M'
  'TimeoutStartSec=45'
)

for directive in "${required_directives[@]}"; do
  if ! grep -Fqx "${directive}" "${unit_path}"; then
    echo "FAIL missing_directive=${directive}"
    exit 1
  fi
done

if systemctl is-enabled 'twirx-egress-worker@.service' >/dev/null 2>&1; then
  echo 'FAIL template_must_not_be_enabled'
  exit 1
fi

if systemctl list-units --state=running --plain --no-legend 'twirx-egress-worker@*.service' | grep -q .; then
  echo 'FAIL egress_worker_is_running'
  exit 1
fi

python3 - /etc/resolv.conf <<'PY'
import ipaddress
import pathlib
import sys

servers = []
for line in pathlib.Path(sys.argv[1]).read_text(encoding="utf-8").splitlines():
    fields = line.split()
    if len(fields) >= 2 and fields[0] == "nameserver":
        servers.append(ipaddress.ip_address(fields[1].split("%", 1)[0]))
if not servers:
    raise SystemExit("FAIL controlled_dns_resolver=missing")
blocked = [str(address) for address in servers if not address.is_global]
if blocked:
    raise SystemExit("FAIL controlled_dns_resolver_in_denied_range=" + ",".join(blocked))
print("PASS controlled_dns_resolvers_are_public=true")
PY

echo 'PASS systemd_unit_verified=true'
echo 'PASS private_range_firewall_directives_present=true'
echo 'PASS template_enabled=false'
echo 'PASS active_workers=0'
