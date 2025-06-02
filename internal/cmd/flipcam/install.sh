#!/usr/bin/env bash

# Make sure you have built caddy using go build -C ./cmd/caddy

set -euo pipefail
IFS=$'\n\t'

source_dir=$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")

visudo --check --file "$source_dir/sudoers.conf"

# Add flipcam group
set +e
if getent group flipcam >/dev/null; then
	set -e
	echo "Group 'flipcam' already exists."
else
	set -e
	echo "Creating group 'flipcam'..."
	groupadd --system flipcam
fi

# Move files
install -m 755 -d /etc/flipcam/

install -m 644 "$source_dir/caddy.json" "{{.CaddyConfPath}}"
install -m 644 "$source_dir/{{.CaddyServiceName}}" "/etc/systemd/system/{{.CaddyServiceName}}"

install -m 644 "$source_dir/dnsmasq.conf" "{{.DnsmasqConfPath}}"
install -m 644 "$source_dir/{{.DnsmasqServiceName}}" "/etc/systemd/system/{{.DnsmasqServiceName}}"

install -m 600 "$source_dir/hostapd.conf" "{{.HostapdConfPath}}"
install -m 644 "$source_dir/{{.HostapdServiceName}}" "/etc/systemd/system/{{.HostapdServiceName}}"

install -m 640 "$source_dir/sudoers.conf" "/etc/sudoers.d/50_flipcam"
install -m 640 -g polkitd "$source_dir/{{.PolkitFilename}}" "/etc/polkit-1/rules.d/50_flipcam.rules"

systemctl daemon-reload

# Install caddy
install -m 755 "$source_dir/../cmd/caddy/caddy" "/usr/local/bin/caddy"

set +e
if getent group caddy >/dev/null; then
	set -e
	echo "Group 'caddy' already exists."
else
	set -e
	echo "Creating group 'caddy'..."
	groupadd --system caddy
fi

# User
set +e
if id -u caddy >/dev/null 2>&1; then
	set -e
	echo "User 'caddy' already exists. Ensuring properties are set..."
	usermod \
		--gid caddy \
		--home /var/lib/caddy \
		--shell /usr/sbin/nologin \
		--comment "Caddy web server" \
		caddy
else
	set -e
	echo "Creating user 'caddy'..."
	useradd --system \
		--gid caddy \
		--create-home \
		--home-dir /var/lib/caddy \
		--shell /usr/sbin/nologin \
		--comment "Caddy web server" \
		caddy
fi

if [ ! -d /var/lib/caddy ]; then
  echo "Creating home directory /var/lib/caddy..."
  mkdir -p /var/lib/caddy
fi

echo "Setting ownership and permissions for /var/lib/caddy..."
chown caddy:caddy /var/lib/caddy
chmod 700 /var/lib/caddy
