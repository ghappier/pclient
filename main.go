// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The main package for the Prometheus server executable.
package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof" // Comment this line to disable pprof endpoint.
	"net/url"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ghappier/pclient/client"
	//"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/model"
	//"github.com/prometheus/common/version"
	//"golang.org/x/net/context"

	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/storage/remote"
)

func main() {
	os.Exit(Main())
}

var (
	/*
		configSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "prometheus",
			Name:      "config_last_reload_successful",
			Help:      "Whether the last configuration reload attempt was successful.",
		})
		configSuccessTime = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "prometheus",
			Name:      "config_last_reload_success_timestamp_seconds",
			Help:      "Timestamp of the last successful configuration reload.",
		})
	*/
	Receiver  = flag.String("receiver", "http://127.0.0.1:9990/receive", "接收数据的地址，格式：http://127.0.0.1:9990/receive")
	MetricNum = flag.Int("metrics", 100, "每台机器的指标数量")
	VmNum     = flag.Int("vms", 50, "机器的数量")
	Numps     = flag.Int("numps", 500, "发送多少批数据后睡眠1秒钟")
	Batchs    = flag.Int("batchs", 5, "发送多少批数据")
	Interval  = flag.Int("interval", 60000, "间隔多少毫秒发送一批数据")
)

/*
func init() {
	prometheus.MustRegister(version.NewCollector("prometheus"))
}
*/
// Main manages the startup and shutdown lifecycle of the entire Prometheus server.
func Main() int {

	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	if flag.NArg() == 0 {
		flag.PrintDefaults()
	}

	log.Infoln("Starting prometheus")
	log.Infoln("Build context")

	var (
		reloadables []Reloadable
	)

	remoteAppender := &remote.Writer{}
	reloadables = append(reloadables, remoteAppender)

	if err := reloadConfig(reloadables...); err != nil {
		log.Errorf("Error loading config: %s", err)
		return 1
	}

	metrics, hosts, err := client.InitData(*MetricNum, *VmNum)
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}
	fmt.Println(time.Now(), " : 开始发送数据")
	begin := time.Now()
	wgroup := new(sync.WaitGroup)
	for i := 0; i < *Batchs; i++ {
		wgroup.Add(1)
		go func(index int, total int, wg *sync.WaitGroup) {
			defer wg.Done()
			start := time.Now()
			fmt.Println(time.Now(), " : 开始发送第", index, "/", total, "批数据")
			samples := client.GenSamples(metrics, hosts)
			for i, ss := range samples {
				for _, sample := range ss {
					remoteAppender.Append(sample)
				}
				if i != 0 && i%*Numps == 0 {
					time.Sleep(1 * time.Second)
				}
			}
			end := time.Now()
			fmt.Println(time.Now(), " : 第", index, "批数据发送完成，耗时：", end.Sub(start))
		}(i+1, *Batchs, wgroup)
		if i < *Batchs-1 {
			time.Sleep(time.Duration(*Interval) * time.Millisecond)
		}
	}
	wgroup.Wait()
	complete := time.Now()
	fmt.Println(time.Now(), " : 所有数据发送完成，耗时：", complete.Sub(begin))

	defer remoteAppender.Stop()

	//prometheus.MustRegister(configSuccess)
	//prometheus.MustRegister(configSuccessTime)

	log.Info("See you next time!")
	return 0
}

// Reloadable things can change their internal state to match a new config
// and handle failure gracefully.
type Reloadable interface {
	ApplyConfig(*config.Config) error
}

func reloadConfig(rls ...Reloadable) (err error) {
	log.Infof("Loading configuration")
	/*
		defer func() {
			if err == nil {
				configSuccess.Set(1)
				configSuccessTime.Set(float64(time.Now().Unix()))
			} else {
				configSuccess.Set(0)
			}
		}()
	*/
	//murl, merr := url.Parse("http://127.0.0.1:9990/receive")
	murl, merr := url.Parse(*Receiver)
	if merr != nil {
		fmt.Printf("url parse error:%s\n", merr.Error())
	}
	remoteConfig := new(config.RemoteWriteConfig)
	remoteConfig.URL = &config.URL{murl}
	remoteConfig.RemoteTimeout = model.Duration(30 * time.Second)

	remoteConfigList := make([]*config.RemoteWriteConfig, 0, 1)
	remoteConfigList = append(remoteConfigList, remoteConfig)

	var conf = new(config.Config)
	conf.RemoteWriteConfigs = remoteConfigList

	failed := false
	for _, rl := range rls {
		if err := rl.ApplyConfig(conf); err != nil {
			log.Error("Failed to apply configuration: ", err)
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("one or more errors occurred while applying the new configuration")
	}
	return nil
}
