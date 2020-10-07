# Monitoring

Metrics are used to measure the performance of the `pinta-scheduler`. While it is unconventional to use collected metrics as part of a feedback loop to improve scheduling decisions, it is unlikely to cause any draw backs either. At the currently stage, we are using metrics for performance monitoring purposes only. 

We propose to use `Prometheus` to collect metrics reported from `pinta-scheduler` in terms of service demands including arrival rates, service time, service requirements, and resource utilization. We are also interested in the service overhead which is the time and resources it takes for a job to get ready. We are interested in collecting these information to measure the performance of scheduling algorithms for different machine learning jobs.

## Metrics and Implementation
We are mostly interested in collecting summary which is `histogram` data. We can use `counters` for simple indicators.

### PintaJob
For pinta jobs, we are collecting:
 - Queueing Time: time spent from job created till getting scheduled.
 - Schedule Time: time that scheduler uses to filter and prioritize nodes for job pods.
 - OverHead Time: time from getting scheduled to available state, which includes image pulling and pod initiating.
 - Service Time: running time from job available to done. (some pods in the job might start early while others wait)
 - Arrival Rate: At per hour (or other intervals) period for jobs to arrive.
 - Job CPU, Mem, GPU requirements and usages
 - Node CPU, Mem, GPU usages %
 

## Tracking scheduler, scheduling algorithm, and job information
We are interested in tracking `scheduler`, `scheduling algorithm`, and `job name` to compare the differences between using different methods. The `pinta-scheduler` needs to be compared against other gang schedulers as well as the default scheduling algorithm as a baseline model. Each scheduling algorithms used for each job would allow a fair comparison in terms of their performance. Each of these information would be included as a label per trace to store in the prometheus database. 
> It is important to note that the metric cardinality should be below 10 and therefore we should not have more than 10 labels for the entire system.