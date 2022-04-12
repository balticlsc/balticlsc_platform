# BalticLSC platform components

## Namespace controller
_Path: /namespace-controller_

__Description__

Namespaces are global; thus, conflicts can happen. If someone tries to create a namespace that already
exist, he will get an error back. A suggestion would be to require users to add a prefix to all namespaces
that are created.
When creating a namespace via kubernetes API (kubectl), it is mandatory to add annotation
(field.cattle.io/projectId: [clusterId]:[projectId]) to the spec which binds this new namespace to the users
project. The problem is that it is possible to create a namespace without this annotation, but the user is
then not allowed to access it. One solution that is to delete all namespaces that are created without any
annotation. Another and better solution would be to add this annotation upon namespace creation.
To solve the above, a software component called “Namespace Controller” needs to be implemented. Its
task will be:
1. To watch for namespace events and make sure that namespaces gets associated with the user’s
project
www.balticlsc.eu
2. Remove annotation references to wrong projects, or just make sure an error is returned to user
This controller must be installed in the “Rancher cluster” to be able to access all information needed.
Note: When creating namespace via Rancher API or Rancher UI then it is put inside the correct project.


## Deployment scripts
_Path: /deployment_

__Description__

Scripts to deploy instance of BalticLSC platform written in ansible.

## Nvidia drivers
_Path: /nvidia-drivers_

__Description__

To be able to use Nvidia GPUs drivers needs to be installed on every worker node that has GPUs. The driver
should be as new as possible to support the newest CUDA libraries. Note that the driver supports all older versions
of CUDA.


## Incress rule conflict controller
_Path: /ingress-rule-conflict-controller_

__Description__

In Kubernetes it is possible to create two ingresses with the same host rule (ex: service.foo.com). What
happens with nginx ingress controller is that the second (conflicting) rule is ignored. However, the
second ingress object is created, and it is not possible to see that provisioning failed. And when you are
not in control of the whole Kubernetes cluster (shared cluster), it is not possible to see the conflicting
rule. 
This controller adds missing ingress rule conflict handling to Kubernetes API which
return error upon conflict.

## Tests
Path: /tests

### Boogeyman
_Path: /tests/boogeyman_

__Description__

Test tool to stress test kubernetes cluster. For instance kill random pods at random times.

## Reporting framework (Operator metering)
_Path: /upstream/operator-metering_

__Description__

The cluster monitoring data in Prometheus is very thorough and is only stored for a set number of hours.
For this reason, it is needed to continuously store user utilization for every user. To do this there is a
framework called Operator Metering that can be used.
Operator Metering is part of the Operator Framework from CoreOS. Operator Framework is an open-
source toolkit designed to manage Kubernetes Operators in an effective, automated and scalable way.

This is a git submodule pointing to the upstream operator-metering repository.

## Operator metering custom reports
_Path: /resource-monitoring/operator-metering_

__Description__

Operator Metering framework enables custom usage reporting derived from monitoring data that can be
used for our purposes. It is suggested to generate reports in following intervals:
* Hourly
* Daily
* Monthly

The usage breakdown reports should be by namespace, which can then be used to summarize
project/user utilization for a given period. In addition, it would be nice to summarize the namespace
reports into user reports. The reports should be created on following metrics:
* CPU request – Amount of reserved CPU
* CPU usage – Real CPU usage (recorded)
* Memory request – Amount of reserved memory
* Memory usage – Real memory usage (recorded)
* Storage request – Amount of reserved storage
* Storage usage – Real storage usage
* GPU request – Number of reserved GPUs of different types

## Prometheus metering custom metrics
_Path: /resource-monitoring/prometheus_

__Description__

For smaller clusters it can be overkill to use operator-metering which can handle clusters of thousands of nodes. In these cases prometheus monitoring is enough. However, to be able to deliver the reports per project
that we want (see "Operator metering custom reports"-section), some custom metrics are needed. This directory
contain whats needed to run only prometheus as reporting source.
