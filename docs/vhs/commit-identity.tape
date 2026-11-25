# Empty commits in two repositories, each using the mask's git identity.
Output docs/gifs/commit-identity.gif
Set Shell bash
Set FontFamily "FiraCode Nerd Font"
Set FontSize 17
Set Width 1100
Set Height 340
Set Padding 24
Set Theme "Catppuccin Mocha"
Set TypingSpeed 50ms
Set CursorBlink false

Hide
Type 'source docs/vhs/demo.sh'
Enter
Sleep 500ms
Type 'clear'
Enter
Sleep 300ms
Show
Sleep 500ms

Type 'cd dev/huddlesurety'
Enter
Sleep 1400ms

Type 'git commit --allow-empty -m "Add rate limit headers"'
Enter
Sleep 1200ms

Type 'git log -1 --format="%h %an <%ae>"'
Enter
Sleep 2000ms

Type 'cd ../fuelmap'
Enter
Sleep 1400ms

Type 'git commit --allow-empty -m "Cache fuel prices"'
Enter
Sleep 1200ms

Type 'git log -1 --format="%h %an <%ae>"'
Enter
Sleep 2200ms
