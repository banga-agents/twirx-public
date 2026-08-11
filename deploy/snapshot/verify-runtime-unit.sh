#!/usr/bin/env bash
set -euo pipefail

unit_path=${1:-/etc/systemd/system/twirx-snapshot.service}

if [[ ! -f "$unit_path" ]]; then
  echo "FAIL unit_missing=$unit_path"
  exit 1
fi

systemd-analyze verify "$unit_path"

required_directives=(
  'User=twirx-snapshot'
  'NoNewPrivileges=true'
  'ProtectSystem=strict'
  'SocketBindAllow=tcp:8091'
  'SocketBindDeny=any'
  'IPAddressDeny=any'
  'IPAddressAllow=localhost'
  'CPUQuota=200%'
  'MemoryHigh=1536M'
  'MemoryMax=2048M'
  'MemorySwapMax=0'
  'IOWeight=1'
  'TasksMax=64'
  'LimitNOFILE=256'
)

for directive in "${required_directives[@]}"; do
  if ! grep -Fqx "$directive" "$unit_path"; then
    echo "FAIL missing_directive=$directive"
    exit 1
  fi
done

if systemctl is-enabled twirx-snapshot.service >/dev/null 2>&1; then
  echo 'FAIL candidate_service_must_not_be_enabled'
  exit 1
fi

if systemctl is-active twirx-snapshot.service >/dev/null 2>&1; then
  echo 'FAIL candidate_service_must_not_be_running'
  exit 1
fi

echo 'PASS systemd_unit_verified=true'
echo 'PASS loopback_network_policy_present=true'
echo 'PASS resource_limits_present=true'
echo 'PASS candidate_enabled=false'
echo 'PASS candidate_running=false'
