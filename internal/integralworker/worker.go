package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/JuanJoseLL/distributed_integrals/rpc/integrales"
	"github.com/Knetic/govaluate"
)

func main() {
	client := integrales.NewMasterProtobufClient("http://localhost:8080", &http.Client{})
	for {
		task, err := client.RegisterWorker(context.Background(), &integrales.WorkerInfo{})
    fmt.Println("Task: ", task)
		if err != nil {
			// handle error (e.g., sleep and retry)
			time.Sleep(time.Second * 10)
			continue
		}

		result := calculateIntegral(task)
    fmt.Println("Result: ", result)
		_, err = client.SubmitResult(context.Background(), &integrales.IntegralResult{Result: result})
		if err != nil {
      fmt.Println("Error: ", err)
			// handle error
		}
	}
}

func calculateIntegral(task *integrales.Task) float64 {
	a := task.LowerBound     // lower bound of the integral
	b := task.UpperBound     // upper bound of the integral
	n := int(task.NumIntervals) // number of intervals
	function := task.Function // the function to integrate, represented as a string

	// Calculate the width of each subinterval
	deltaX := (b - a) / float64(n)

	// Summation variable
	sum := 0.0

	// Calculate the Riemann Sum using the midpoint method
	for i := 0; i < n; i++ {
		mid := a + (float64(i)+0.5)*deltaX
		fMid := evaluateFunction(function, mid) // evaluate the function at the midpoint
		sum += fMid * deltaX
	}

	return sum
}

func evaluateFunction(function string, x float64) float64 {
	expression, err := govaluate.NewEvaluableExpression(function)
	if err != nil {
		log.Fatalf("Error creating expression: %v", err)
	}

	parameters := make(map[string]interface{}, 1)
	parameters["x"] = x

	result, err := expression.Evaluate(parameters)
	if err != nil {
		log.Fatalf("Error evaluating expression: %v", err)
	}

	// Assuming the function is expected to return a float64.
	if val, ok := result.(float64); ok {
		return val
	} else {
		log.Fatalf("Evaluation did not return a float64: %v", result)
		return 0
	}
}
