# RHEL Custom ISO Builder

Tool to bake a Kickstart file into a RHEL install ISO, producing a fully unattended installer image (no manual install steps at boot).

## What it does

`build-iso.go` wraps `mkksiso` (from `lorax`) and `ksvalidator` (from `pykickstart`):

1. Checks `mkksiso`/`ksvalidator` are installed; installs them via `dnf` if missing.
2. Validates the source ISO and kickstart file exist.
3. Runs `ksvalidator` against the kickstart file — fails fast on syntax errors.
4. Runs `mkksiso --ks <ks.cfg> <source.iso> <output.iso>`, which injects the kickstart file into the ISO and sets it to auto-load at boot (`inst.ks=cdrom:/ks.cfg`).
5. Confirms the output ISO exists and prints its size.

Net effect: boot the resulting ISO on bare metal/VM → install runs unattended using `ks.cfg`.

**Note on tooling choice:** this is a linear sequence of shell-outs (`command -v`, `dnf`, `ksvalidator`, `mkksiso`, `stat`) with no real business logic, a Bash script would be the natural/idiomatic tool for this but Go is used here purely because I'm more comfortable in it. Functionally equivalent, just more verbose (compiled binary, explicit error handling, etc).

## Prerequisites

| Requirement | Notes |
|---|---|
| RHEL/Fedora/CentOS host (or container) | Needs `dnf` and root/sudo |
| `sudo` privileges | Required for package install and `mkksiso` (loopback mount) |
| `lorax` package | Provides `mkksiso` |
| `pykickstart` package | Provides `ksvalidator` |
| Go ≥ 1.18 (build-time only) | `go build -o build-iso build-iso.go` |
| **RHEL DVD ISO — not the Boot/netinstall ISO** | See below, this is critical |
| Valid RHEL subscription / registered system for the *installed* machine | Not for building — for `dnf` on the resulting install if you pull anything outside BaseOS/AppStream |

Install build deps manually if you don't want the Go tool to `sudo dnf install` for you:
```bash
sudo dnf install -y lorax pykickstart
```

## Why it must be the DVD ISO (not Boot ISO)

RHEL ships two ISO variants:

- **Boot ISO** (`rhel-9.x-x86_64-boot.iso`) — minimal, ~2GB, stages packages over the network from configured repos during install. If the target machine isn't registered/subscribed at install time, `%packages` resolution fails or hangs.
- **DVD ISO** (`rhel-9.x-x86_64-dvd.iso`) — full, ~14GB, ships **BaseOS + AppStream repos on the media itself**. The `cdrom` install source in `ks.cfg` reads directly from these embedded repos — no network/subscription required to resolve packages.

**Use the DVD ISO as `sourceISO`.** With `cdrom` as the install source, only packages present in BaseOS/AppStream on that DVD are installable. Anything else (EPEL, CRB/PowerTools, custom repos) will cause the install to fail outright, or silently need `subscription-manager register` + `attach` — which won't happen unattended in `%post` on a factory image unless credentials are baked in (bad practice) or a satellite/proxy is present.

**Rule of thumb:** if a package isn't in `dnf repolist` on a stock BaseOS+AppStream registered system, don't put it in `%packages` unless you also add a repo (see below).

## Packages used in `ks.cfg`

```
@^server-product-environment   # package group: Server product environment (base group for RHEL Server installs)
nginx                          # AppStream
openssh-server                 # BaseOS
tmux                           # BaseOS
```

All three are in BaseOS/AppStream on the DVD — no extra repo needed for this config as-is.

## Kickstart file breakdown (`ks.cfg`)

| Section | Purpose |
|---|---|
| `#version=RHEL9` | Tells `ksvalidator`/anaconda which syntax/schema to validate against |
| `cdrom` | Install source = the DVD media itself (see above) |
| `lang`, `keyboard`, `timezone` | Locale config, no prompts at install |
| `network --bootproto=dhcp --onboot=on` | NIC auto-config, brought up at install and on boot |
| `rootpw`, `user` | Accounts — **plaintext passwords, change before building**; consider `--iscrypted` with a pre-hashed value instead |
| `zerombr`, `clearpart --all --initlabel` | Wipes target disk's partition table — destructive, no confirmation |
| `part /boot/efi`, `/boot`, `/`, `swap`, `/home` | Manual partition layout (UEFI + XFS + swap) |
| `bootloader --location=mbr` | GRUB2 target — mismatched with `/boot/efi` above; see Known Issues |
| `firstboot --disable` | Skip the post-install setup wizard |
| `eula --agreed` | Skip EULA prompt |
| `reboot` | Auto-reboot after install completes |
| `%packages ... %end` | Package/group selection, see table above |
| `%post ... %end` | Runs inside installed system chroot after packages land; here: `systemctl enable nginx sshd` |

## Known issues / gotchas in the provided `ks.cfg`

- `bootloader --location=mbr` is BIOS/legacy syntax but a `part /boot/efi` (UEFI) partition is also defined — pick one target firmware type consistently, or `mkksiso`/anaconda may ignore one of them depending on how the ISO is booted (BIOS vs UEFI).
- Plaintext passwords in the file — anyone with the ISO has root. Use `--iscrypted` with `openssl passwd -6` output, or inject at build time from a secrets store.
- No `%packages` `--nocore` / `--excludedocs` flags — fine for this use case, mentioned for reference.
- `ksvalidator` only checks syntax, not whether every listed package/group actually resolves against your chosen ISO's repos — a typo'd package name here will only surface as a failure during `mkksiso`'s actual install run, not at validation time.

## Usage

```bash
go build -o build-iso build-iso.go
sudo ./build-iso /path/to/rhel-9.x-x86_64-dvd.iso ks.cfg custom.iso
```

## Or Simply

```bash
go run build-iso.go /path/to/rhel-9.x-x86_64-dvd.iso ks.cfg custom.iso
```

Args: `<source.iso> <ks.cfg> [output.iso=custom.iso]`