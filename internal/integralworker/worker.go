package main

import (
	"context"
	"fmt"
	"log"
	"math"
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
      fmt.Println("Error: ", err)
			time.Sleep(time.Second * 10)
			continue
		}
    initialTime := time.Now() 
		result := calculateIntegral(task)
    endTme := time.Now()
    fmt.Println("Time: ", endTme.Sub(initialTime))
    fmt.Println("Result: ", result)
		_, err = client.SubmitResult(context.Background(), &integrales.IntegralResult{Result: result})
		if err != nil {
      fmt.Println("Error: ", err)
			
		}
	}
}

var customFunctions = map[string]govaluate.ExpressionFunction{
  "exp": func(args ...interface{}) (interface{}, error) {
      if len(args) != 1 {
          return nil, fmt.Errorf("exp() expects exactly one argument")
      }
      switch arg := args[0].(type) {
      case float64:
          return math.Exp(arg), nil
      default:
          return nil, fmt.Errorf("exp() expects a numerical argument")
      }
  },
  "sin": func(args ...interface{}) (interface{}, error) {
      if len(args) != 1 {
          return nil, fmt.Errorf("sin() expects exactly one argument")
      }
      switch arg := args[0].(type) {
      case float64:
          return math.Sin(arg), nil
      default:
          return nil, fmt.Errorf("sin() expects a numerical argument")
      }
  },
  "cos": func(args ...interface{}) (interface{}, error) {
      if len(args) != 1 {
          return nil, fmt.Errorf("cos() expects exactly one argument")
      }
      switch arg := args[0].(type) {
      case float64:
          return math.Cos(arg), nil
      default:
          return nil, fmt.Errorf("cos() expects a numerical argument")
      }
  },
  "tanh": func(args ...interface{}) (interface{}, error) {
      if len(args) != 1 {
          return nil, fmt.Errorf("tanh() expects exactly one argument")
      }
      switch arg := args[0].(type) {
      case float64:
          return math.Tanh(arg), nil
      default:
          return nil, fmt.Errorf("tanh() expects a numerical argument")
      }
  },
}

func calculateIntegral(task *integrales.Task) float64 {
	a := task.LowerBound     // lower bound of the integral
	b := task.UpperBound     // upper bound of the integral
	n := int(task.NumIntervals) // number of intervals
	function := task.Function // the function to integrate, represented as a string

  expression, err := govaluate.NewEvaluableExpressionWithFunctions(function, customFunctions)
  if err != nil {
		log.Fatalf("Error creating expression: %v", err)
	}
	// Calculate the width of each subinterval
	deltaX := (b - a) / float64(n)

	// Summation variable
	totalSum := 0.0

  partitions := 2
  sumChan := make(chan float64, partitions)
	// Calculate the Riemann Sum using the midpoint method
	for p := 0; p < partitions; p++ {
		go func(p int) {
			localSum := 0.0
			for i := p * n / partitions; i < (p+1)*n/partitions; i++ {
				mid := a + (float64(i)+0.5)*deltaX
				fMid := evaluateFunction(expression, mid)
				localSum += fMid * deltaX
			}
			sumChan <- localSum
		}(p)
	}
  
	for p := 0; p < partitions; p++ {
		totalSum += <-sumChan
	}


	return totalSum
}

func evaluateFunction(expression *govaluate.EvaluableExpression, x float64) float64 {

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
