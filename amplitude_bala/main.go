package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"amplitude_bala/internal/event"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"runtime"
	"time"
	"context"
	"github.com/a8m/kinesis-producer"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"	
	"github.com/rs/xid"
	
)


var pr *producer.Producer
//Intilializes kpl producer stream
func init(){
	client := kinesis.New(session.New(aws.NewConfig()))
	pr = producer.New(&producer.Config{
		StreamName:   "amplitude-stream",
		BatchCount: 100,
		AggregateBatchSize: 4,
		BacklogCount: 100,
		FlushInterval: 1,
		Client:       client,

	})

	pr.Start()

}

//Sets Respne Headers
func amplitudeResponse(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html;charset=utf-8")
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-allow-methods", "GET, POST")
		w.Header().Set("strict-transport-security", "max-age=15768000")
		next.ServeHTTP(w, r)
	}
}
//Writes error logs
func writeError(w http.ResponseWriter, r *http.Request, error string, code int) {
	log.Printf("Error: %s %s %s %d", r.Method, r.URL, error, code)
	http.Error(w, error, code)
}

// Receieve the data from ELB and move it to kinesis stream.
func handleAmplitude(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
	elbEvent := event.NewELB(r)
	jsonData, err := json.Marshal(elbEvent)

	if err != nil {
		writeError(w, r, "invalid_json", 500)
		return
	}
	
	if len(elbEvent.Body) == 0 {
		http.Error(w, "no_body", 404)
		return
	}
	
	guid := xid.New()
		err2 := pr.Put([]byte(string(jsonData)+"\n"),
 guid.String())
	// err2 := pr.Put(jsonData, guid.String())
	
	if err2 != nil {
		log.Println("error producing", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request){
	w.WriteHeader(http.StatusOK)
 	w.Write([]byte("☄ HTTP status code returned!"))
}
// handles the http requests.
func handleRoot(w http.ResponseWriter, r *http.Request) {
	handleAmplitude(w, r)
}
// handles NotFoundHandler error
func handleError(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, "error", 500)
}
func notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, "not_found", 404)
}

/* handles, MethodNotAllowedHandler ( Configurable Handler to be used when the request
	method does not match the route.)*/
func notAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, "not_allowed", 405)
}

// Basic Server confiuration
func main() {

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	/* Router to route incomming request*/
	r := mux.NewRouter()
	r.HandleFunc("/", amplitudeResponse(handleRoot)).Methods("POST", "GET")
	r.HandleFunc("/health", handleHealth).Methods("POST", "GET")
	r.NotFoundHandler = http.HandlerFunc(amplitudeResponse(notFound))
	r.MethodNotAllowedHandler = http.HandlerFunc(amplitudeResponse(notAllowed))
	// Assigning port and handler for Http Server
	addr := ":8088"
	h := &http.Server{Addr: addr, Handler: r}

	// function to start the server
	go func() {
		log.Printf("starting server....  %v %v \n", runtime.GOMAXPROCS(0), runtime.NumCPU())
		err := h.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("fatal error running the server %v \n", err)
		}
	}()
	s := <-termChan // wait for signal
	log.Printf("ending server.... \n",s)
	pr.Stop()
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	log.Printf("shutdown with timeout: %s\n", timeout)
	err := h.Shutdown(ctx)
	if err != nil {
		log.Printf("error ending server.... %v \n", err)
	}
	log.Printf("done \n")
}
