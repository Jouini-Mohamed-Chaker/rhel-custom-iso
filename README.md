# RHEL Auto-Install ISO

Builds a self-installing RHEL ISO from a Kickstart file.

## Requirements

- Fedora, RHEL, or another RHEL-family distro (CentOS Stream, Rocky, Alma...)
- Go installed, **or** just the pre-built `build-iso` binary (no Go needed then)
- `sudo` access (used to install `lorax`/`pykickstart` if missing)

## Files

- `original-rhel.iso` — official RHEL ISO (you provide, from Red Hat)
- `ks.cfg` — install config (included, edit passwords first)
- `build-iso.go` — builds the automated ISO

## 1. Edit ks.cfg

Change these before building:
```
rootpw --plaintext ChangeMe123!
user --name=admin --groups=wheel --plaintext --password=ChangeMe123!
```

Config summary: separate `/boot`, `/`, `swap`, `/home` · DHCP networking ·
root + `admin` (sudo) user · Server install + nginx, ssh, htop · fr keyboard,
Tunis timezone.

## 2. Build

Run directly with Go:
```bash
go run build-iso.go original-rhel.iso ks.cfg custom-rhel-automated.iso
```

Or build once and reuse the binary (only useful if you already have Go to build it):
```bash
go build -o build-iso build-iso.go
./build-iso original-rhel.iso ks.cfg custom-rhel-automated.iso
```

Installs `lorax`/`pykickstart` if missing, validates `ks.cfg`, then bakes the ISO.

## 3. Use in a VM

Attach `custom-rhel-automated.iso` as the VM's boot/install disk (CD/DVD drive)
and start the VM — the install runs unattended using the settings in `ks.cfg`.

## Note

Passwords in `ks.cfg` are plaintext, fine for local/lab use. For production,
hash them with `openssl passwd -6` and use `--iscrypted`.