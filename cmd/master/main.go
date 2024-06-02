// master
package main

import (

	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	
	"strconv"
	"sync"
	"time"

	"github.com/JuanJoseLL/distributed_integrals/rpc/integrales"
)

type Server struct {
	taskQueue   []integralTask
	results     []float64
	queueLock   sync.Mutex
	resultLock  sync.Mutex
}

type integralTask struct {
	Request  *integrales.Task
	Response chan *integrales.IntegralResult
}

type integralResult struct {
	Result *float64
}



var (
	connections []*integrales.WorkerInfo
	connMutex   sync.Mutex
	taskCompleted bool
	startJob *bool
)

// RegisterWorker allows workers to connect and receive tasks.
func (s *Server) RegisterWorker(ctx context.Context, req *integrales.WorkerInfo) (*integrales.Acknowledgment, error) {
	connMutex.Lock()
	defer connMutex.Unlock()
	connections = append(connections, req)
	fmt.Println("Worker registered", req)
	return &integrales.Acknowledgment{Received: true}, nil
}

// SubmitResult allows workers to return their computed results.
func (s *Server) SubmitResult(ctx context.Context, resp *integrales.IntegralResult) (*integrales.Acknowledgment, error) {
	fmt.Println("Result received")
	s.resultLock.Lock()
	defer s.resultLock.Unlock()
	s.results = append(s.results, resp.Result)
	return &integrales.Acknowledgment{Received: true}, nil
}

func (s *Server) GetTask(ctx context.Context, req *integrales.WorkerInfo) (*integrales.Task, error) {

	s.queueLock.Lock()
	defer s.queueLock.Unlock()
	if len(s.taskQueue) > 0 {
		task := s.taskQueue[0]
		s.taskQueue = s.taskQueue[1:]
		return &integrales.Task{
			LowerBound:   task.Request.LowerBound,
			UpperBound:   task.Request.UpperBound,
			NumIntervals: task.Request.NumIntervals,
			Function:     task.Request.Function,
		}, nil
	}
	return nil, fmt.Errorf("no tasks available")
}

func main() {
		s := &Server{}

		taskCompleted = false

		lowerBoundStr := flag.String("lower", "0", "Lower bound of the integral")
		upperBoundStr := flag.String("upper", "0", "Upper bound of the integral")
		numIntervalsStr := flag.String("intervals", "10000", "Number of intervals for the Riemann sum")
		functionStr := flag.String("function", "sin(x^2)", "Function to integrate")
		

		flag.Parse()

		lowerBound, err := strconv.ParseFloat(*lowerBoundStr, 64)
		if err != nil {
			log.Fatalf("Invalid lower bound: %v", err)
		}

		upperBound, err := strconv.ParseFloat(*upperBoundStr, 64)
		if err != nil {
			log.Fatalf("Invalid upper bound: %v", err)
		}

		numIntervals, err := strconv.Atoi(*numIntervalsStr)
		if err != nil {
			log.Fatalf("Invalid number of intervals: %v", err)
		}

		numIntervals32 := int32(numIntervals)

		

		// Create a new Task with the user's integral
		
		// Add the task to the server's queue
		startJob := false
		go func() {
			for {
				if startJob {
					createTasks(lowerBound, upperBound, numIntervals32, *functionStr, s)
					go s.WaitAndPrintResults()
					break
				}
				time.Sleep(1 * time.Second)
			}
		}()
		http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
			startJob = true
			fmt.Fprintf(w, "Job started\n")
		})


		twirpHandler := integrales.NewMasterServer(s, nil)
		http.Handle(twirpHandler.PathPrefix(), twirpHandler)

		fmt.Println("Listening on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
		
			
}

func createTasks(lowerBound, upperBound float64, numIntervals int32, function string, s *Server){
	fmt.Println("Creating tasks")
	connMutex.Lock()
	numWorkers := len(connections)
	connMutex.Unlock()
	if numWorkers == 0 {
		fmt.Printf("No workers connected\n")
		return
	}
	intervalSize := (upperBound - lowerBound) / float64(numWorkers)
	for i := 0; i < numWorkers; i++ {
		
		taskLowerBound := lowerBound + float64(i)*intervalSize
		taskUpperBound := taskLowerBound + intervalSize
		fmt.Println("Task ", i+1, " lower bound: ", taskLowerBound, " upper bound: ", taskUpperBound)
		task := &integrales.Task{
			
			LowerBound:   taskLowerBound,
			UpperBound:   taskUpperBound,
			NumIntervals: int32(numIntervals),
			Function: function,
		}
		s.queueLock.Lock()
		s.taskQueue = append(s.taskQueue, integralTask{Request: task})
		s.queueLock.Unlock()
		
	
	}

}

func (s *Server) WaitAndPrintResults() {
    for {
		
        s.resultLock.Lock()
		fmt.Printf("Results: %d\n", len(s.results))
		fmt.Println("Tasks: ", len(s.taskQueue))
		if len(s.taskQueue) == 0 {
			fmt.Println("All tasks completed. Results:")
			taskCompleted = true
			resultSum := 0.0
			for _, result := range s.results {
				resultSum += result
				
			}
			
			fmt.Println("Sum: ", resultSum)
			s.resultLock.Unlock()
			break
		}
        s.resultLock.Unlock()
        // Sleep for a while before checking again.
        time.Sleep(10 * time.Second)
    }
}