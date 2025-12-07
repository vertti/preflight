Output cli-demo.gif
Set Shell bash
Set FontSize 16
Set Width 800
Set Height 400
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "preflight cmd node --min 18.0"
Enter
Sleep 2s

Type "preflight env HOME"
Enter
Sleep 2s

Type "preflight cmd node --min 99.0"
Enter
Sleep 2s

Type "preflight env HOME --match '^/nonexistent'"
Enter
Sleep 5s
