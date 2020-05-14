/*********************************************************
File: server.go
Contents: This file contains the API endpoint implementations and related data
*********************************************************/

package server

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RequestStat struct {
	Total   int64 `json:"total"`
	Average int64 `json:"average"`
}

const (
	// URL paths
	HashPath     = "/hash"
	StatsPath    = "/stats"
	ShutdownPath = "/shutdown"

	// Form fields
	PasswordKey = "password"

	// Error messages
	ErrInvalidId     = "Error: Invalid task Id"
	ErrPassword      = "Error: Missing or invalid password"
	ErrShutdown      = "Service is shutting down, request rejected"
	ErrShutdownError = "Server encountered an error while shutting down: %v"

	// Farewell message
	MsgFarewell = "All requests have been processed, terminating service."
	MsgShutdown = "Initiating service shutdown"
	
	// Runtime constants
	ListenPort = 8080
	DelayTime  = 5 * time.Second
)

var (
	// Request counter, incremented for each request, used as request Id
	requestID int64 = 0
	// Results are stored here
	resultMap = make(map[string]string)
	// Mutexes to protect requestId and resultMap
	mtxId, mtxMap sync.Mutex
	// Total time spent processing POST requests
	elapsedTime int64 = 0
	// Server object
	httpServer http.Server
	// Shutdown flag
	bShutdown = false
)

/* method delayAndUpdate()
- Sleep for the required amount of time
- Calculate SHA512 of `pword`
- Put result in resultMap using requestId as key
*/
func delayAndUpdate(requestId string, pword string) {
	// Pause before processing
	time.Sleep(DelayTime)
	
	// Hash the password
	sum := sha512.Sum512([]byte(pword))
	
	// Convert to Base64
	sha := base64.URLEncoding.EncodeToString(sum[:])
	
	// Add to resultMap
	mtxMap.Lock()
	resultMap[requestId] = sha
	mtxMap.Unlock()
	
	log.Printf("Deferred processing completed for request Id %s", requestId)
}

/*
	method doHash()
	Handle POST and GET request for URL path `/hash`
*/
func doHash(w http.ResponseWriter, r *http.Request) {
	// Sorry, not taking any more requests
	if bShutdown {
		http.Error(w, ErrShutdown, http.StatusServiceUnavailable)
		return
	}
	// Keep track of start time
	startTime := time.Now()

	// handle GET and POST requests
	switch r.Method {
	case http.MethodGet:
		// parse out the request Id
		id := strings.TrimPrefix(r.URL.Path, HashPath+"/")
		result := resultMap[id]
		if len(result) > 0 {
			// Output the result
			_, err := fmt.Fprintf(w, result)
			if err != nil {
				log.Printf("Error sending HTTP response: %v", err)
			}
		} else {
			// No entry found for specified key
			http.Error(w, ErrInvalidId, http.StatusBadRequest)
		}
	case http.MethodPost:
		// Get the password from the form
		pw := r.FormValue(PasswordKey)
		if len(pw) == 0 {
			// Password missing
			http.Error(w, ErrPassword, http.StatusBadRequest)
			return
		}
		// Increment request Id
		mtxId.Lock()
		requestID++
		num := strconv.FormatInt(requestID, 10)
		mtxId.Unlock()
		
		// Fire off goroutine to do the work
		go delayAndUpdate(num, pw)
		
		// return the requestId
		_, err := fmt.Fprintf(w, num)
		if err != nil {
			log.Printf("Error sending HTTP response: %v", err)
		}

		// Update statistics
		mtxId.Lock()
		elapsedTime += time.Since(startTime).Microseconds()
		mtxId.Unlock()

		log.Printf("Request %s posted for deferred processing", num)

	default:
		// We only support GET and POST methods here
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

/*
	method getStats()
	Return a JSON object with the current statistics
*/
func getStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		// Only GET method is supported
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	// If we're shutting down we will not accept requests
	if bShutdown {
		http.Error(w, "Service is shutting down, request rejected", http.StatusServiceUnavailable)
		return
	}

	// get current counts
	stats := RequestStat{
		Total:   0,
		Average: 0,
	}

	mtxId.Lock()
	stats.Total = requestID
	et := elapsedTime
	mtxId.Unlock()

	// calculate average if count != 0
	if stats.Total != 0 {
		stats.Average = et / stats.Total
	}

	// Serialize and return the stats
	jtext, _ := json.Marshal(stats)
	_, err := w.Write(jtext)
	if err != nil {
		log.Printf("Error returning statistics: %v", err)
	}
}

/*
	method doShutdown
	- Set the shutdown flag to stop accepting new requests
	- Wait for any pending requests to complete
	- Shut down the HTTP server
*/
func doShutdown(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodGet {
		// Only GET method is supported
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	log.Printf(MsgShutdown)
	bShutdown = true

	/* 	Wait for all requests to complete.  This is done by comparing the
	number of requests accepted to the number of entries in resultMap.
	When they match all requests have been processed
	*/
	for {
		mtxMap.Lock()
		count := len(resultMap)
		mtxMap.Unlock()
		if int64(count) == requestID {
			break
		}
		// Wait a short time before checking again
		time.Sleep(100 * time.Millisecond)
	}

	// Respond with a farewell message
	_, err := fmt.Fprintf(w, MsgFarewell)
	if err != nil {
		log.Printf("Failed to send farewell message: %v", err)
	}

	log.Printf("Shutdown: All tasks completed")
	// Give the server some time to send the request, then terminate it
	go func() {
		time.Sleep(1 * time.Second)
		err := httpServer.Shutdown(nil)
		if err != http.ErrServerClosed {
			log.Printf(ErrShutdownError, err)
		}
	}()
}

/*
	method HandleRequests()
	Server's only export.  This sets up the handlers and deploys a listening server.
*/
func StartServer(port int) {
	http.HandleFunc(HashPath, doHash)
	http.HandleFunc(HashPath+"/", doHash)
	http.HandleFunc(StatsPath, getStats)
	http.HandleFunc(ShutdownPath, doShutdown)
	httpServer = http.Server{Addr: ":" + strconv.Itoa(port)}
	log.Fatal(httpServer.ListenAndServe())
}
