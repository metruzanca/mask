# Each mask points git at its own SSH key.
Output docs/gifs/ssh-key.gif
Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 17
Set Width 1100
Set Height 280
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set CursorBlink false

Hide
Type 'source docs/vhs/demo.sh'
Enter
Sleep 500ms
Type 'cd dev/huddlesurety'
Enter
Sleep 300ms
Type 'clear'
Enter
Sleep 300ms
Show
Sleep 500ms

Type 'echo "$GIT_SSH_COMMAND" | sed "s|$HOME|~|"'
Enter
Sleep 1800ms

Type 'cd ../fuelmap'
Enter
Sleep 1600ms

Type 'echo "$GIT_SSH_COMMAND" | sed "s|$HOME|~|"'
Enter
Sleep 2200ms
