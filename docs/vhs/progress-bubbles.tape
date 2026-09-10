Output docs/vhs/progress-bubbles.gif
Require kubexporter
Set Shell bash
Set FontSize 16
Set Width 1400
Set Height 900
Set Padding 20
Set Framerate 12
Set PlaybackSpeed 1.5
Set TypingSpeed 40ms
Hide
Type "export PS1='$ '" Enter
Type "cd $(mktemp -d)" Enter
Type "clear" Enter
Show
Type "kubexporter --progress bubbles -w2" Enter
Sleep 20s
