/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"log"
	"sync"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

// run is a function to run the client, and upon completion, it will record the latency.
func run(address *string, defaultName *string, output chan<- *hdrhistogram.Histogram) {
	// Set up a connection to the server.
	conn, err := grpc.Dial(*address+":80",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := *defaultName
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use HdrHistogram to measure the latency.
	h := hdrhistogram.New(1, 1000000, 3)

	// Call SayHello repetitively and measure the latency.
	count := 0
	for i := 0; i < 100000; i++ {
		start := time.Now()
		r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
		if err != nil {
			break
		}
		elapsed := time.Since(start)
		count++

		// Check the response.
		if r.GetMessage() != "Hello "+name {
			log.Fatalf("unexpected response: %v", r.Message)
		}

		h.RecordValue(elapsed.Nanoseconds())
	}

	output <- h
}

func main() {
	address := flag.String("address", "localhost:50051", "address of the server")
	defaultName := flag.String("name", "world", "name to greet")
	clients := flag.Int("clients", 1, "number of clients")

	flag.Parse()

	// Construct a WaitGroup to wait for all clients to finish.
	var wg sync.WaitGroup

	// Construct a channel to receive the latency from each client.
	output := make(chan *hdrhistogram.Histogram, *clients)

	// Run the clients.
	for i := 0; i < *clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(address, defaultName, output)
		}()
	}

	// Wait for all clients to finish.
	wg.Wait()

	// Read the latency from each client and merge them.
	h := hdrhistogram.New(1, 1000000, 3)
	count := 0
	for i := 0; i < *clients; i++ {
		hist := <-output
		h.Merge(hist)
		count += int(hist.TotalCount())
	}

	log.Printf("Latency: avg=%v, p50=%v, p95=%v, p99=%v, p999=%v, p9999=%v, max=%v",
		time.Duration(h.Mean()), time.Duration(h.ValueAtQuantile(50)),
		time.Duration(h.ValueAtQuantile(95)), time.Duration(h.ValueAtQuantile(99)),
		time.Duration(h.ValueAtQuantile(99.9)), time.Duration(h.ValueAtQuantile(99.99)),
		time.Duration(h.Max()))

	log.Printf("Number of requests: %v", count)
}
