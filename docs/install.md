# Install

Pick whichever line matches how you already install tools. All of them give you
the same `relio` binary.

> Every version and its binaries: **[github.com/soyagvs/relio/releases](https://github.com/soyagvs/relio/releases)**

## Which install should I use?

If the Homebrew tap is published and available to you, use Homebrew. If it is not
available yet, use `go install` or download the release binary manually.

Check what your shell is running with:

```bash
which relio
relio version
```

## Homebrew (macOS / Linux) — recommended when the tap is available

Install:

```bash
brew tap soyagvs/tap
brew install relio
```

Update:

```bash
brew update
brew upgrade relio
```

The formula downloads the official prebuilt binaries from this repo's
[GitHub Releases](https://github.com/soyagvs/relio/releases) — it does not build
from source.

## With Go — current fallback when Homebrew is not available

Install or update to the latest published version:

```bash
go install github.com/soyagvs/relio@latest
```

Go installs the binary into `$GOBIN` when set, otherwise into `$GOPATH/bin`
(usually `~/go/bin`). Make sure that directory is on your `PATH`.

## Manual — from GitHub Releases

Grab the archive for your platform from the
[latest release](https://github.com/soyagvs/relio/releases/latest), check it
against `checksums.txt`, and drop the binary on your `PATH`:

```bash
VER=1.5.0                      # the release you want
OS=darwin; ARCH=arm64          # darwin|linux|windows  +  amd64|arm64
curl -fsSLO "https://github.com/soyagvs/relio/releases/download/v${VER}/relio_${VER}_${OS}_${ARCH}.tar.gz"
curl -fsSLO "https://github.com/soyagvs/relio/releases/download/v${VER}/checksums.txt"
sha256sum -c --ignore-missing checksums.txt
tar -xzf "relio_${VER}_${OS}_${ARCH}.tar.gz" relio
sudo mv relio /usr/local/bin/
```

Windows: download `relio_<ver>_windows_amd64.zip` and put `relio.exe` on your
`PATH`.

## From source

```bash
git clone https://github.com/soyagvs/relio
cd relio
go build -o relio .          # ./relio
# or: make install           # builds to ~/.cargo/bin/relio
```

Building requires **Go 1.22+**. At runtime Relio needs the `git` binary on
`PATH`.

## Shell completion

Tab-completion for subcommands and flags, courtesy of Cobra — nothing extra to
install beyond the `relio` binary itself.

```bash
# bash (needs bash-completion installed)
relio completion bash > /etc/bash_completion.d/relio        # or ~/.local/share/bash-completion/completions/relio

# zsh — first run: echo "autoload -U compinit; compinit" >> ~/.zshrc
relio completion zsh > "${fpath[1]}/_relio"

# fish
relio completion fish > ~/.config/fish/completions/relio.fish

# PowerShell — add to your $PROFILE
relio completion powershell >> $PROFILE
```

`relio completion <shell> --help` explains the exact setup for that shell in
more detail (loading it in the current session vs. persisting it).
