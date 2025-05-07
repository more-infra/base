# [v0.9.11] 2025-05-07
## Ehance
- [kv] 新增struct类型inline标签支持
---
# [v0.9.10] 2025-04-28
## Ehance
- [kv] 新增time_fmt的tag，Marshal, Unmarshal均可支持time.Time类型的自定义截取
---
# [v0.9.9] 2025-02-28
## Enhance
- [tjson] add time type supported
- [event] add event package
- [observer] add observer package
---
# [v0.9.8] 2024-10-26
## Enhance
- [tjson] add package with json custom type for json.Unmarshal 
---
# [v0.9.7] 2024-08-26
## Enhance
- [kv package] add supports map input for mapper.Marshal
---
# [v0.9.6] 2024-06-29
## BugFix
- [scheduler package] fix test case TestScheduler FAIL
## Enhance
- [runner package] Runner add Context() method for returning the controlling context of the Runner.
---
# [v0.9.5] 2024-05-24
## Breaking Change
- [trigger package] replace handler param callback by queue.Buffer for receiving the elements batch pack by trigger.
## Bugs
- [kv package] fix marshal failed for time.Time, time.Duration type marshal as struct
## Features
- [kv package] add MapperMarshaller interface for custom defined data marshaller
## Enhance
- [base] OriginalError use recursion check.
- [base] Error() format adjust, use "\n" split each fields.
- [kv package] ObjectToStruct map type support empty key
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