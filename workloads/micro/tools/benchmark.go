package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"cs.utexas.edu/zjia/faas-micro/client"
	"cs.utexas.edu/zjia/faas-micro/utils"
	"github.com/montanaflynn/stats"
)

var FLAGS_faas_gateway string
var FLAGS_fn_prefix string
var FLAGS_num_users int
var FLAGS_concurrency int
var FLAGS_duration int
var FLAGS_percentages string
var FLAGS_bodylen int
var FLAGS_rand_seed int
var FLAGS_microbenchmark_type string
var FLAGS_loop_internal bool

var FLAGS_record_length int
var FLAGS_latency_bucket_lower int
var FLAGS_latency_bucket_upper int
var FLAGS_latency_bucket_granularity int

const (
	Append        string = "append"
	AppendAndRead string = "appendAndRead"
	Read          string = "read"
)

func init() {
	flag.StringVar(&FLAGS_faas_gateway, "faas_gateway", "127.0.0.1:8081", "")
	flag.StringVar(&FLAGS_fn_prefix, "fn_prefix", "", "")
	flag.IntVar(&FLAGS_num_users, "num_users", 1000, "")
	flag.IntVar(&FLAGS_concurrency, "concurrency", 1, "")
	flag.IntVar(&FLAGS_duration, "duration", 10, "")
	flag.IntVar(&FLAGS_rand_seed, "rand_seed", 23333, "")
	flag.StringVar(&FLAGS_microbenchmark_type, "microbenchmark_type", "append", "")
	flag.BoolVar(&FLAGS_loop_internal, "loop_internal", false, "")

	flag.IntVar(&FLAGS_record_length, "record_length", 1024, "")
	flag.IntVar(&FLAGS_latency_bucket_lower, "latency_bucket_lower", 300, "")            // microsec
	flag.IntVar(&FLAGS_latency_bucket_upper, "latency_bucket_upper", 10000, "")          //microsec
	flag.IntVar(&FLAGS_latency_bucket_granularity, "latency_bucket_granularity", 10, "") //microsec

	rand.Seed(int64(FLAGS_rand_seed))
}

func buildAppendToLogRequest() utils.JSONValue {
	return utils.JSONValue{}
}

func buildAppendToLogLoopRequest() utils.JSONValue {
	record := utils.CreateRecord(FLAGS_record_length)
	fmt.Printf("Length of record is %d", len(record))
	return utils.JSONValue{
		"record":                     record,
		"loop_duration":              FLAGS_duration,
		"latency_bucket_lower":       FLAGS_latency_bucket_lower,
		"latency_bucket_upper":       FLAGS_latency_bucket_upper,
		"latency_bucket_granularity": FLAGS_latency_bucket_granularity,
	}
}

func buildReadFromLogRequest() utils.JSONValue {
	return utils.JSONValue{}
}

func buildAppendToAndReadFromLogRequest() utils.JSONValue {
	return utils.JSONValue{}
}

func printFnResult(fnName string, duration time.Duration, results []*utils.FaasCall) {
	total := 0
	succeeded := 0
	latencies := make([]float64, 0, 128)
	for _, result := range results {
		if result.FnName == FLAGS_fn_prefix+fnName {
			total++
			if result.Result.Success {
				succeeded++
			}
			if result.Result.StatusCode == 200 {
				d := result.Result.Duration
				latencies = append(latencies, float64(d.Microseconds()))
			}
		}
	}
	if total == 0 {
		return
	}
	failed := total - succeeded
	fmt.Printf("[%s]\n", fnName)
	fmt.Printf("Throughput: %.1f requests per sec\n", float64(total)/duration.Seconds())
	if failed > 0 {
		ratio := float64(failed) / float64(total)
		fmt.Printf("Failed requests: %d (%.2f%%)\n", failed, ratio*100.0)
	}
	if len(latencies) > 0 {
		median, _ := stats.Median(latencies)
		p99, _ := stats.Percentile(latencies, 99.0)
		fmt.Printf("Latency: median = %.3fms, tail (p99) = %.3fms\n", median/1000.0, p99/1000.0)
	}
}

func internalBenchmark() {
	log.Printf("[INFO] Deploy loop functions. Function mode: %s. Concurrency: %d. Duration: %d", FLAGS_microbenchmark_type, FLAGS_concurrency, FLAGS_duration)
	c := 0
	client := client.NewFaasClient(FLAGS_faas_gateway, FLAGS_concurrency)
	for c < FLAGS_concurrency {
		switch FLAGS_microbenchmark_type {
		case Append:
			client.AddJsonFnCall(FLAGS_fn_prefix+"AppendToLogLoop", buildAppendToLogLoopRequest())
		default:
			fmt.Printf("Unknown argument %s", FLAGS_microbenchmark_type)
			break
		}
		c++
	}
	faasCalls := client.WaitForResults()

	for _, faasCall := range faasCalls {
		if faasCall.Result.Success {
			appendLoopOutput := faasCall.Result.AppendLoopOutput
			fmt.Printf("Calls Total: %d", appendLoopOutput.Calls)
			fmt.Printf("Calls Success: %d", appendLoopOutput.Success)
			fmt.Printf("Calls Failure: %d", appendLoopOutput.Calls-appendLoopOutput.Success)
			fmt.Printf("Throughput: %.2f [Op/s] using %d bytes records", appendLoopOutput.Throughput, FLAGS_record_length)
		}
	}
}

func clientBenchmark() {
	log.Printf("[INFO] Start running client for %d seconds with concurrency of %d", FLAGS_duration, FLAGS_concurrency)
	client := utils.NewFaasClient(FLAGS_faas_gateway, FLAGS_concurrency)
	startTime := time.Now()
	for {
		if time.Since(startTime) > time.Duration(FLAGS_duration)*time.Second {
			break
		}
		switch FLAGS_microbenchmark_type {
		case Append:
			client.AddJsonFnCall(FLAGS_fn_prefix+"AppendToLog", buildAppendToLogRequest())
		case Read:
			client.AddJsonFnCall(FLAGS_fn_prefix+"ReadFromLog", buildReadFromLogRequest())
		case AppendAndRead:
			client.AddJsonFnCall(FLAGS_fn_prefix+"AppendToAndReadFromLog", buildAppendToAndReadFromLogRequest())
		default:
			fmt.Printf("Unknown argument %s", FLAGS_microbenchmark_type)
			break
		}
	}
	results := client.WaitForResults()
	elapsed := time.Since(startTime)
	fmt.Printf("Benchmark runs for %v, %.1f request per sec\n", elapsed, float64(len(results))/elapsed.Seconds())

	printFnResult("AppendToLog", elapsed, results)
	printFnResult("ReadFromLog", elapsed, results)
	printFnResult("AppendToAndReadFromLog", elapsed, results)
}

func main() {
	flag.Parse()
	if FLAGS_loop_internal {
		internalBenchmark()
	} else {
		clientBenchmark()
	}
}
