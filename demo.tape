# EdControls CLI Demo
# Run with: vhs demo.tape

Output assets/demo.gif

Set FontSize 14
Set Width 900
Set Height 600
Set Theme "Catppuccin Mocha"
Set Padding 20

# Title
Type "# EdControls CLI Demo"
Enter
Sleep 1s

# Show version
Type "ec --version"
Enter
Sleep 1.5s

# Show help
Type "ec --help"
Enter
Sleep 3s

# Check who we are
Type "ec whoami"
Enter
Sleep 2s

# List projects
Type "ec projects list"
Enter
Sleep 3s

# List tickets for a project
Type "ec tickets list nl_ithelden_9c62eecc-b16b-459d-9e3c-fa76e5ab0cc2 -l 5"
Enter
Sleep 3s

# Get ticket details
Type "ec tickets get 6D8A21"
Enter
Sleep 3s

# List audits
Type "ec audits list nl_ithelden_9c62eecc-b16b-459d-9e3c-fa76e5ab0cc2 -l 5"
Enter
Sleep 3s

# Show templates
Type "ec templates list nl_ithelden_9c62eecc-b16b-459d-9e3c-fa76e5ab0cc2 -l 5"
Enter
Sleep 3s

# Final message
Type "# Install: brew tap dutchview/tap && brew install ec"
Enter
Sleep 2s
