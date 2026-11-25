# Environment variables attached to a mask appear and disappear with it.
Output docs/gifs/env-vars.gif
Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 17
Set Width 1100
Set Height 320
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

Type 'echo "AWS_PROFILE=${AWS_PROFILE:-unset}"'
Enter
Sleep 1600ms

Type 'cd ../fuelmap'
Enter
Sleep 1600ms

Type 'echo "AWS_PROFILE=${AWS_PROFILE:-unset}"'
Enter
Sleep 1600ms

Type 'cd ..'
Enter
Sleep 1600ms

Type 'echo "AWS_PROFILE=${AWS_PROFILE:-unset}"'
Enter
Sleep 2200ms
