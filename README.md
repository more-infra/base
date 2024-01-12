# more-infra/base

[![Github CI](https://github.com/more-infra/base/actions/workflows/go.yml/badge.svg?branch=main&event=pull_request)]()
[![Go Reference](https://pkg.go.dev/badge/github.com/more-infra/base.svg)](https://pkg.go.dev/github.com/more-infra/base)  

The basic structures and design modes which could be used by projects.  
It includes many typical and common data structures,functions for basic using.

## Packages

| name | description                                                                                                     |
|:----|:----------------------------------------------------------------------------------------------------------------|
|error|basic struct for error interface which has more information wrapped                                             |
|runner|background goroutine controller with channel sign and sync.WaitGroup                                            |
|status|work status controller such as starting,started,stopping,stopped                                                |
|element|item container supports safe thread operations and provides simple features database used, such as index,search |

## Development

Now, this project is in developing stage, so the code is changing frequently.
A new version will be published every week at Friday.