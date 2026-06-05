# Chameleon showcase — "one color to theme them all".
#
# Two acts that make the whole pitch:
#   Act 1  four unrelated tools (git · npm · docker · kubectl), ONE theme —
#          same prompt glyph, same glyph-and-color-per-state grammar, same
#          aligned columns. They read as siblings.
#   Act 2  change the theme with a single env var — every adapter follows.
#
# Reproducible & hermetic: `git` is real; npm/docker/kubectl are stubbed by the
# committed fixtures in demo/fixtures/ (see their headers) so the recording needs
# no network, no daemon, and no cluster. Chameleon's rendering is 100% real —
# only the upstream machine-output it parses is fixed.
#
# Regenerate from the repo root:
#   go build -o chameleon .
#   vhs demo/showcase.tape          # -> demo/showcase.gif

Output demo/showcase.gif

Require git

Set Shell "bash"
Set FontSize 18
Set Width 900
Set Height 820
Set Padding 30
Set Theme "TokyoNight"
Set TypingSpeed 45ms

# ---- hidden setup -------------------------------------------------------------
# Put the fixtures (npm/docker/kubectl) and the freshly built `chameleon` first
# on PATH, force color through the pipe-less VHS pty, and build a tiny git repo
# with one change of every kind (added/deleted/renamed/modified/untracked).
Hide
Type `export PATH="$PWD/demo/fixtures:$PWD:$PATH"` Enter
Type `export CHAMELEON_FORCE_COLOR=1` Enter
Type `export PS1='$ '` Enter
Type `D=$(mktemp -d); cd "$D"` Enter
Type `git init -q -b main && git config user.email demo@chameleon.dev && git config user.name demo` Enter
Type `printf 'package main\n' > app.go; echo legacy > old.txt; echo 'port: 8080' > config.yml` Enter
Type `git add app.go old.txt config.yml && git commit -qm init` Enter
Type `echo 'new file' > parser.go && git add parser.go && git rm -q old.txt && git mv config.yml settings.yml && echo 'more code' >> app.go && echo scratch > debug.tmp` Enter
Type `clear` Enter
Show

# ---- Act 1: one theme (tokyonight, the default), four unrelated tools ---------
Sleep 800ms
Type `# one theme — git, npm, docker, kubectl all drawn as siblings` Sleep 400ms Enter
Sleep 500ms
Type `chameleon git status` Sleep 400ms Enter
Sleep 1600ms
Type `chameleon npm outdated` Sleep 400ms Enter
Sleep 1600ms
Type `chameleon docker ps` Sleep 400ms Enter
Sleep 1600ms
Type `chameleon kubectl get pods` Sleep 400ms Enter
Sleep 2600ms

# ---- Act 2: change the theme ONCE — every adapter follows ---------------------
Hide
Type `clear` Enter
Show
Type `# change the theme once — everything follows` Sleep 400ms Enter
Sleep 500ms
Type `export CHAMELEON_THEME=gruvbox` Sleep 400ms Enter
Sleep 700ms
Type `chameleon git status` Sleep 400ms Enter
Sleep 1400ms
Type `chameleon npm outdated` Sleep 400ms Enter
Sleep 1400ms
Type `chameleon docker ps` Sleep 400ms Enter
Sleep 1400ms
Type `chameleon kubectl get pods` Sleep 400ms Enter
Sleep 3200ms
