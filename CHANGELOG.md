# [v0.9.5]
## Breaking Change
- [trigger package] replace handler param callback by queue.Buffer for receiving the elements batch pack by trigger.
---
# [v0.9.4] 2024-02-09
## Enhance
- [kv package] add supports convert map[string]interface{} to object.
---
# [v0.9.3] 2024-02-02
## Features
- [kv package] convert struct object with tag defined to a map[string]interface{} like json/yaml Marshal do.
# [v0.9.2] 2024-01-26
## Features
- [varfmt package] format string with variable by custom syntax and variable value provider
- [scheduler package] dynamic goroutine pool for executing tasks, which could be controlled by custom options
- [util/algo package] algorithm utility such as md5, base64, zlib
---
# [v0.9.1] 2024-01-19
## Documents
- README add go doc, github-ci, release, go report, MIT-License badges
- element, status package comment for go doc format fixed
## Features
- [queue package] chan with dynamic capacity
- [reactor package] reactor design mode for resolving sync locking and concurrent controlling.
- [chanpool package] do select for multiple channels which is ambiguous
- [trigger package] pack input elements stream to batch by controlling params or function
- [values package] strings matcher with multiple regex, wildcard pre-built
---
# [v0.9.0] 2024-01-12
## Features
- error basic struct used for typical error return value
- [status package] background goroutine loop task controller with channel sign and sync.WaitGroup
- [element package] item container that support simple database features.