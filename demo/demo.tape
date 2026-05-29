# Chameleon demo recording.
#
# Regenerate the GIF with:
#   go build -o chameleon .
#   vhs demo/demo.tape
#
# VHS is a Charm dev tool used only to record docs — it is NOT a runtime
# dependency. The Chameleon binary itself pulls in zero Charm packages.

Output demo/demo.gif

Require git

Set Shell bash
Set FontSize 20
Set Width 1040
Set Height 600
Set Padding 28
Set Theme "TokyoNight"
Set TypingSpeed 120ms
Set PlaybackSpeed 1.0

# ---- hidden setup: a throwaway repo with a bit of everything ----
Hide
Type "REPO=$PWD; export PATH=$REPO:$PATH; export CHAMELEON_FORCE_COLOR=1; export PS1='\\$ '" Enter
Type "D=$(mktemp -d); cd $D" Enter
Type "git init -q --bare origin.git && git clone -q origin.git work && cd work" Enter
Type "git config user.email demo@chameleon && git config user.name demo" Enter
Type "printf 'package main\\n' > app.go && echo legacy > old.txt && git add -A && git commit -qm init" Enter
Type "git push -q -u origin HEAD:main" Enter
Type "echo a > feature.go && git add feature.go && git commit -qm feature" Enter
Type "echo b > cache.go && git add cache.go && git commit -qm cache" Enter
Type "echo new > parser.go && git add parser.go" Enter
Type "git rm -q old.txt" Enter
Type "echo more >> app.go" Enter
Type "echo scratch > debug.tmp" Enter
Type "cp -r $REPO/themes . && echo themes/ >> .git/info/exclude" Enter
Type "clear" Enter
Show

# ---- the actual demo ----
Sleep 1500ms
Type "chameleon git status" Sleep 1200ms Enter
Sleep 7s
