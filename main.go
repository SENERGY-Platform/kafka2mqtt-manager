/*
 * Copyright 2021 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"flag"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg"
	"github.com/SENERGY-Platform/kafka2mqtt-manager/pkg/config"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
)

func main() {
	time.Sleep(5 * time.Second) //wait for routing tables in cluster

	confLocation := flag.String("config", "config.json", "configuration file")
	flag.Parse()

	conf, err := config.Load(*confLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	wg, err := lib.Start(conf, ctx)
	if err != nil {
		debug.PrintStack()
		log.Fatal("FATAL:", err)
	}

	var shutdownTime time.Time
	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		sig := <-shutdown
		log.Println("received shutdown signal", sig)
		shutdownTime = time.Now()
		cancel()
	}()

	wg.Wait()
	log.Println("Shutdown complete, took", time.Since(shutdownTime))
}
