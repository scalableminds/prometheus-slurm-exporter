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
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"io/ioutil"
	"os/exec"
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
	Jobs []SqueueJob `json:"jobs"`
}

type SqueueJob struct {
	Account   string             `json:"account"`
	Id        int                `json:"job_id"`
	Name      string             `json:"name"`
	Resources SqueueJobResources `json:"job_resources"`
	State     []string           `json:"job_state"`
	Nodes     string             `json:"nodes"`
	Partition string             `json:"partition"`
	GroupId   int                `json:"group_id"`
	GroupName string             `json:"group_name"`
	UserId    int                `json:"user_id"`
	UserName  string             `json:"user_name"`
}

type SqueueJobResources struct {
	Cpus  int `json:"cpus"`
	Nodes struct {
		Allocation []SqueueJobResourcesAllocation `json:"allocation"`
	} `json:"nodes"`
}

type SqueueJobResourcesAllocation struct {
	Memory struct {
		Allocated int `json:"allocated"`
	} `json:"memory"`
}

func InstrumentJobs() SqueueResult {
	jobs := JobsData()
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
	labels := []string{
		"account",
		"job_id",
		"name",
		"cpus",
		"memory",
		"state",
		"nodes",
		"partition",
		"group_id",
		"group_name",
		"user_id",
		"user_name",
	}
	return &JobsCollector{
		jobs: prometheus.NewDesc("slurm_jobs", "Description of running Slurm jobs", labels, nil),
	}
}

func (jc *JobsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- jc.jobs
}

func (jc *JobsCollector) Collect(ch chan<- prometheus.Metric) {
	jm := InstrumentJobs()
	for _, job := range jm.Jobs {
		allocatedMemory := 0
		for _, allocation := range job.Resources.Nodes.Allocation {
			allocatedMemory += allocation.Memory.Allocated
		}
		labels := []string{
			job.Account,
			fmt.Sprintf("%d", job.Id),
			job.Name,
			fmt.Sprintf("%d", job.Resources.Cpus),
			fmt.Sprintf("%d", allocatedMemory),
			fmt.Sprintf("%s", job.State),
			job.Nodes,
			job.Partition,
			fmt.Sprintf("%d", job.GroupId),
			job.GroupName,
			fmt.Sprintf("%d", job.UserId),
			job.UserName,
		}
		ch <- prometheus.MustNewConstMetric(jc.jobs, prometheus.GaugeValue, 1.0, labels...)
	}
}
