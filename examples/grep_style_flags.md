# Grep-Style Context Flags

Bashistdb now supports grep-style context flags for the -A, -B, and -C options. This means you can use either traditional or grep-style syntax:

## Traditional Format
```bash
bashistdb -A 5 "git commit"    # Show 5 lines after each match
bashistdb -B 10 "error"        # Show 10 lines before each match  
bashistdb -C 3 "docker"        # Show 3 lines before and after each match
```

## Grep-Style Format (NEW)
```bash
bashistdb -A5 "git commit"     # Show 5 lines after each match
bashistdb -B10 "error"         # Show 10 lines before each match
bashistdb -C3 "docker"         # Show 3 lines before and after each match
```

## Mixed Usage
You can mix both styles and combine with other flags:
```bash
bashistdb -A5 -B 10 -g "deploy"       # -A5 (grep-style) with -B 10 (traditional)
bashistdb -C3 -R "^git.*push$"        # Context search with regex
bashistdb -A10 -unique "docker build" # Show unique commands with context
```

## Implementation Details
The preprocessing happens transparently before flag parsing, converting:
- `-A5` → `-A 5`
- `-B10` → `-B 10`
- `-C3` → `-C 3`

This maintains backward compatibility while adding the convenience of grep-style syntax.