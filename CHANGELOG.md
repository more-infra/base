# [v0.9.1] 2024-01-19
## Documents
- README add go doc, github-ci badges
- element, status package comment for go doc format fixed
## Features
- [queue package] chan with dynamic capacity
- [reactor package] reactor design mode for resolving sync locking and concurrent controlling.
- [chanpool package] do select for multiple channels which is ambiguous
---
# [v0.9.0] 2024-01-12
## Features
- error basic struct used for typical error return value
- [status package] background goroutine loop task controller with channel sign and sync.WaitGroup
- [element package] item container that support simple database features. 