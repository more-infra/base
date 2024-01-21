# more-infra/base

[![Github CI](https://github.com/more-infra/base/actions/workflows/testing.yml/badge.svg)]()
[![Go Report Card](https://goreportcard.com/badge/github.com/more-infra/base)](https://goreportcard.com/report/github.com/more-infra/base)
[![Release](https://img.shields.io/github/v/release/more-infra/base.svg?style=flat-square)](https://github.com/more-infra/base)
[![Go Reference](https://pkg.go.dev/badge/github.com/more-infra/base.svg)](https://pkg.go.dev/github.com/more-infra/base)
[![License: MIT](https://img.shields.io/badge/License-MIT-orange)](https://opensource.org/licenses/MIT)

The basic structures and design modes which could be used by projects.  
It includes many typical and common data structures,functions for basic using.

## Packages

| name     | description                                                                                                          |
|:---------|:---------------------------------------------------------------------------------------------------------------------|
| error    | basic struct for error interface which has more information wrapped                                                  |
| runner   | background goroutine controller with channel sign and sync.WaitGroup                                                 |
| status   | work status controller such as starting,started,stopping,stopped                                                     |
| element  | item container supports safe thread operations and provides simple features database used, such as index,search      |
| queue    | chan with dynamic capacity                                                                                           |
| reactor  | reactor design mode for resolving sync locking and concurrent controlling, which is similar to event loop processing |
| mcontext | put multiple contexts into one which implements the context.Context interface                                        |
| chanpool | do select for multiple channels which is ambiguous                                                                   |
| trigger  | pack input elements stream to batch by controlling params or function                                                |
| values   | strings matcher with multiple regex and wildcard pre-built                                                           |
| varfmt   | format string with variable by custom syntax and variable value provider                                             |

## Development

Now, this project is in developing stage, so the code is changing frequently.
A new version will be published every week at Friday.