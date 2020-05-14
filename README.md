# JumpCloud Interview Assignment
This repository contains demo code that implements the functionality specified by the JumpCloud interview assignment document.  
Specifically it is a ReST service that implements the follow API:

API Endpoint|HTTP Method|Description
------------|-----------|------------
/hash | POST | Post a request with a form field `password` with a string value.  The request will be queued for deferred processing and the API will return a task Id that can be used to fetch the results asynchronously 
/hash/task_id| GET | Fetch the results of a queued task.  If the task Id is invalid or the task has not completed this API will return an HTTP `Bad Request` (400) status
/shutdown|GET|Gracefully shut down the service.  Wait for any pending tasks to complete then shut down the service and exit.  Any requests received while shutdown is in process will be failed with HTTP status `Service Unavailable` (503)

Calling any of these APIs with the wrong HTTP method will result in an HTTP error status of `Method not allowed` (405) 

## Building (requires Go 1.14)
* Clone the source - `git clone https://github.com/jameadows/JumpCloud.git`
* CD into `JumpCloud/hash_pass` directory
* Run `go build main.go`

## Running
Typing `./main` will run the server on the default listening port 8080.  The port value can be specified on the command line, e.g. `./main 1234` will run the service listening on port 1234.  Port number must be within range 1024 < port < 65536.
