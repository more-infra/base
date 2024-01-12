# more-infra/base
The basic structures and design modes which could be used by projects.  
It includes many typical and common data structures,functions for basic using. 

## Packages
- **error** &emsp;&emsp;&emsp;&emsp;&emsp;&emsp; basic struct for error interface which has more information wrapped  
- **runner** &emsp;&emsp;&emsp;&emsp;background goroutine controller with channel sign and sync.WaitGroup
- **status** &emsp;&emsp;&emsp;&emsp;&emsp;work status controller such as starting,started,stopping,stopped
- **element** &emsp; item container supports safe thread operations and provides simple features database used, such as index,search
