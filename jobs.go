/* Copyright 2025 Scalableminds

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>. */


package main

import (
	"os/exec"
	"encoding/json"
	"log"
	"fmt"
	"io/ioutil"
	"github.com/prometheus/client_golang/prometheus"
)

func JobsData() []byte {
	cmd := exec.Command("squeue", "--json")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	out, _ := ioutil.ReadAll(stdout)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	return out
}

type SqueueResult struct {
	jobs []SqueueJob
}

type SqueueJob struct {
	account string
	id int `json:"job_id"`
	name string
	resources SqueueJobResources `json:"job_resources"`
	state []string `json:"job_state"`
	nodes string
	partition string
	groupId int `json:"group_id"`
	groupName string `json:"group_name"`
	userId int `json:"user_id"`
	userName string `json:"user_name"`
	memoryPerNode SqeueueMemoryPerNode `json:"memory_per_node"`
}

type SqueueJobResources struct {
	cpus int
}

type SqeueueMemoryPerNode struct {
	number int
}

func InstrumentJobs() SqueueResult {
	jobs := JobsData();
	var result SqueueResult
	if err := json.Unmarshal(jobs, &result); err != nil {
		log.Fatal(err)
	}
	return result
}

type JobsCollector struct {
	jobs *prometheus.Desc
}

func NewJobsCollector() *JobsCollector {
	labels := []string {
		"account",
		"job_id",
		"name",
		"cpus",
		"memory_per_node",
		"state",
		"nodes",
		"partition",
		"group_id",
		"group_name",
		"user_id",
		"user_name",
	};
	return &JobsCollector {
		jobs: prometheus.NewDesc("slurm_jobs", "Description of running Slurm jobs", labels, nil),
	}
}

func (jc *JobsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- jc.jobs
}

func (jc *JobsCollector) Collect(ch chan<- prometheus.Metric) {
	jm := InstrumentJobs()
	for _, job := range jm.jobs {
		labels := []string{
			job.account,
			fmt.Sprintf("%s", job.id),
			job.name,
			fmt.Sprintf("%s", job.resources.cpus),
			fmt.Sprintf("%s", job.memoryPerNode.number),
			fmt.Sprintf("%s", job.state),
			job.nodes,
			job.partition,
			fmt.Sprintf("%d", job.groupId),
			job.groupName,
			fmt.Sprintf("%d", job.userId),
			job.userName,
		}
		ch <- prometheus.MustNewConstMetric(jc.jobs, prometheus.GaugeValue, 1.0, labels...)
	}
}
