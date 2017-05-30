package client

import (
	"errors"
	"math/rand"

	"strconv"
	"time"

	"github.com/prometheus/common/model"
)

//初始化数据
//参数metricNum:每台机器的指标数量
//参数vmNum:机器的数量
//返回值分别为：指标切片、host切片、错误
func InitData(metricNum int, vmNum int) ([]string, []string, error) {
	var metrics []string = make([]string, 0, metricNum)
	var hosts []string = make([]string, 0, vmNum)
	if metricNum < 1 {
		return nil, nil, errors.New("每台机器的指标数量必须大于0")
	} else if metricNum > MetricNamesLen {
		return nil, nil, errors.New("每台机器的指标数量不能大于" + strconv.Itoa(MetricNamesLen))
	} else {
		metrics = MetricNames[0:metricNum]
	}
	if vmNum < 1 {
		return nil, nil, errors.New("机器的数量必须大于0")
	}
	for i := 0; i < vmNum; i++ {
		index := 0
		hosts = append(hosts, Hostnames[index]+strconv.Itoa(i+1))
		index++
		index = vmNum % HostnamesLen
	}
	return metrics, hosts, nil
}

func GenSamples(metrics []string, hosts []string) []model.Samples {
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	metricNum := len(metrics)
	vmNum := len(hosts)
	n := metricNum * vmNum
	var ps int = 100 //每批数据最多包含的指标数量
	num := n / ps
	mod := n % ps
	var count int //总共分成多少批数据
	if mod == 0 {
		count = num
	} else {
		count = num + 1
	}
	result := make([]model.Samples, 0, count)
	samples := make(model.Samples, 0, ps)
	sampleCount := 0
	for i := 0; i < vmNum; i++ {
		for j := 0; j < metricNum; j++ {
			samples = append(samples, &model.Sample{
				Metric: model.Metric{
					model.MetricNameLabel: model.LabelValue(metrics[j]), //指标名称
					model.JobLabel:        "job1",
					model.InstanceLabel:   "instance1",
					model.QuantileLabel:   "0.25",
					"hostname":            model.LabelValue(hosts[i]),
					"domain":              "qq",
					"containerid":         "container_a",
					"env_location":        "dgg",
					"cpu":                 "cpu1",
					"disk":                "disk1",
				},
				Value:     model.SampleValue(rd.Float64()),
				Timestamp: model.Time(time.Now().Unix()),
			})
			sampleCount++
			if sampleCount >= ps {
				result = append(result, samples)
				samples = make(model.Samples, 0, ps)
			}
		}
	}
	if len(samples) > 0 {
		result = append(result, samples)
	}
	return result
}
