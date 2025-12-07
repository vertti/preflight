Output demo.gif
Set Shell bash
Set FontSize 16
Set Width 900
Set Height 550
Set Theme "Catppuccin Mocha"
Set Padding 20

Type "# Show what we're building"
Enter
Sleep 500ms

Type "cat demo/Dockerfile"
Enter
Sleep 4s

Type "# Run the build"
Enter
Sleep 500ms

Type "docker build --progress=plain -f demo/Dockerfile ."
Enter
Sleep 10s
