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

// RegisterWorker allows workers to connect and receive tasks.
func (s *Server) RegisterWorker(ctx context.Context, req *integrales.WorkerInfo) (*integrales.Task, error) {
	fmt.Println("Worker registered")
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

// SubmitResult allows workers to return their computed results.
func (s *Server) SubmitResult(ctx context.Context, resp *integrales.IntegralResult) (*integrales.Acknowledgment, error) {
	fmt.Println("Result received")
	s.resultLock.Lock()
	defer s.resultLock.Unlock()
	s.results = append(s.results, resp.Result)
	return &integrales.Acknowledgment{Received: true}, nil
}

func main() {
	s := &Server{}
	
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
    task := &integrales.Task{
		LowerBound:   lowerBound,
        UpperBound:   upperBound,
        NumIntervals: numIntervals32,
        Function: *functionStr,
        // Fill in the other fields as necessary
    }

    // Add the task to the server's queue
    s.taskQueue = append(s.taskQueue, integralTask{Request: task})

    go s.WaitAndPrintResults()


	twirpHandler := integrales.NewMasterServer(s, nil)
	fmt.Println("Listening on :8080")
	http.ListenAndServe(":8080", twirpHandler)
}

func (s *Server) WaitAndPrintResults() {
    for {
        s.resultLock.Lock()
        if len(s.results) == len(s.taskQueue) {
            fmt.Println("All tasks completed. Results:")
            for i, result := range s.results {
                fmt.Printf("Task %d: %f\n", i+1, result)
            }
            s.resultLock.Unlock()
            break
        }
        s.resultLock.Unlock()
        // Sleep for a while before checking again.
        time.Sleep(1 * time.Second)
    }
}