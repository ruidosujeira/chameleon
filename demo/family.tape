# Chameleon "one theme, two tools" recording — the thesis proof.
#
# Regenerate with:
#   go build -o chameleon .
#   vhs demo/family.tape          # → demo/family.gif
#
# Shows `git status` and `npm outdated` back to back: same palette, same
# glyph-and-color-per-state model, same column discipline — visual siblings.

Output demo/family.gif

Require git
Require npm

Set Shell bash
Set FontSize 20
Set Width 1080
Set Height 760
Set Padding 28
Set Theme "TokyoNight"
Set TypingSpeed 110ms

# ---- hidden setup: a git repo with changes AND an npm project with outdated deps ----
Hide
Type "REPO=$PWD; export PATH=$REPO:$PATH; export CHAMELEON_FORCE_COLOR=1; export PS1='\\$ '" Enter
Type "D=$(mktemp -d); cd $D" Enter
Type "git init -q -b main && git config user.email demo@chameleon && git config user.name demo" Enter
Type "echo 'node_modules/' > .gitignore && echo 'package.json' >> .gitignore && echo 'package-lock.json' >> .gitignore" Enter
Type "echo 'package main' > app.go && echo legacy > old.txt && git add app.go old.txt .gitignore && git commit -qm init" Enter
Type "echo new > parser.go && git add parser.go && git rm -q old.txt && echo more >> app.go && echo scratch > debug.tmp" Enter
Type "npm init -y >/dev/null 2>&1 && npm install --silent --no-audit --no-fund chalk@4.0.0 semver@7.3.0 is-number@6.0.0 >/dev/null 2>&1 && echo SETUP_DONE" Enter
Sleep 45s
Type "clear" Enter
Show

# ---- the actual demo: two tools, one theme ----
Sleep 1200ms
Type "chameleon git status" Sleep 900ms Enter
Sleep 3s
Type "chameleon npm outdated" Sleep 900ms Enter
Sleep 6s
