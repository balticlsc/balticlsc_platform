# Start presto cli in presto-coordinator: presto-cli --server localhost:8080 --catalog hive --schema default 
# use metering
# show tables

select "timestamp", labels['namespace'] as namespace, labels['pod'] as pod, sum(amount * "timeprecision") AS total_gpu_request, sum(amount * "timeprecision") AS avg_gpu_request from datasource_metering_pod_gpu_request group by "timestamp", labels['namespace'], labels['pod'] order by namespace ASC LIMIT 10;

# group by namespace:
select "timestamp", labels['namespace'] as namespace, sum(amount * "timeprecision") AS total_gpu_request, sum(amount * "timeprecision") AS avg_gpu_request from datasource_metering_pod_gpu_request group by "timestamp", labels['namespace'] order by timestamp ASC;


# group by project:
select datasource_metering_pod_gpu_request.timestamp as timestamp, datasource_metering_pod_gpu_request.labels['namespace'] as namespace, datasource_metering_pod_gpu_request.labels['pod'] as pod, datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] as projectid from datasource_metering_pod_gpu_request,datasource_metering_projectid_namespace_map where datasource_metering_pod_gpu_request.timestamp=datasource_metering_projectid_namespace_map.timestamp and datasource_metering_pod_gpu_request.labels['namespace'] LIKE datasource_metering_projectid_namespace_map.labels['namespace'] LIMIT 5;

select datasource_metering_pod_gpu_request.timestamp as timestamp, datasource_metering_pod_gpu_request.labels['namespace'] as namespace, datasource_metering_pod_gpu_request.labels['pod'] as pod, datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] as projectid from datasource_metering_pod_gpu_request,datasource_metering_projectid_namespace_map where datasource_metering_pod_gpu_request.timestamp=datasource_metering_projectid_namespace_map.timestamp and datasource_metering_pod_gpu_request.labels['namespace'] LIKE datasource_metering_projectid_namespace_map.labels['namespace'] group by datasource_metering_pod_gpu_request.timestamp,datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] LIMIT 5;

select datasource_metering_pod_gpu_request.timestamp, datasource_metering_pod_gpu_request.labels['namespace'], datasource_metering_pod_gpu_request.labels['pod'], datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] from datasource_metering_pod_gpu_request,datasource_metering_projectid_namespace_map where datasource_metering_pod_gpu_request.timestamp=datasource_metering_projectid_namespace_map.timestamp and datasource_metering_pod_gpu_request.labels['namespace'] LIKE datasource_metering_projectid_namespace_map.labels['namespace'] group by datasource_metering_projectid_namespace_map.timestamp,datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] LIMIT 5;


select datasource_metering_pod_gpu_request.timestamp, datasource_metering_pod_gpu_request.labels['namespace'] as namespace, datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] as projectid, sum(datasource_metering_pod_gpu_request.amount * datasource_metering_pod_gpu_request.timeprecision) AS total_gpu_request from datasource_metering_pod_gpu_request,datasource_metering_projectid_namespace_map where datasource_metering_pod_gpu_request.timestamp=datasource_metering_projectid_namespace_map.timestamp and datasource_metering_pod_gpu_request.labels['namespace'] LIKE datasource_metering_projectid_namespace_map.labels['namespace'] group by datasource_metering_pod_gpu_request.timestamp,datasource_metering_pod_gpu_request.labels['namespace'],datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] order by timestamp ASC LIMIT 5;

# 
        timestamp        |   namespace    |    projectid    | total_gpu_request
-------------------------+----------------+-----------------+-------------------
 2019-10-29 08:23:00.000 | sdl-jupyterhub | c-vqbq2:p-t49mt |             120.0
 2019-10-29 08:23:00.000 | do-mining      | c-vqbq2:p-5bs8m |             120.0
 2019-10-29 08:24:00.000 | sdl-jupyterhub | c-vqbq2:p-t49mt |             120.0
 2019-10-29 08:24:00.000 | do-mining      | c-vqbq2:p-5bs8m |             120.0
 2019-10-29 08:25:00.000 | do-mining      | c-vqbq2:p-5bs8m |             120.0

select datasource_metering_pod_gpu_request.timestamp,datasource_metering_pod_gpu_request.labels['pod'] as pod, datasource_metering_pod_gpu_request.labels['node'] as node, datasource_metering_pod_gpu_request.labels['namespace'] as namespace, datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] as projectid, sum(datasource_metering_pod_gpu_request.amount * datasource_metering_pod_gpu_request.timeprecision) AS gpu_request_seconds from datasource_metering_pod_gpu_request,datasource_metering_projectid_namespace_map where datasource_metering_pod_gpu_request.timestamp=datasource_metering_projectid_namespace_map.timestamp and datasource_metering_pod_gpu_request.labels['namespace'] LIKE datasource_metering_projectid_namespace_map.labels['namespace'] group by datasource_metering_pod_gpu_request.timestamp,datasource_metering_pod_gpu_request.labels['pod'],datasource_metering_pod_gpu_request.labels['node'],datasource_metering_pod_gpu_request.labels['namespace'],datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] order by timestamp ASC LIMIT 5;

        timestamp        |             pod              |    node     |   namespace    |    projectid    | gpu_request_seconds 
-------------------------+------------------------------+-------------+----------------+-----------------+---------------------
 2019-10-29 08:23:00.000 | miner-5dfdf75864-c7v6h       | p01r03srv07 | do-mining      | c-vqbq2:p-5bs8m |                60.0 
 2019-10-29 08:23:00.000 | miner-5dfdf75864-gcp4d       | p01r03srv04 | do-mining      | c-vqbq2:p-5bs8m |                60.0 
 2019-10-29 08:23:00.000 | jupyter-rickard-2ebrannvall  | p01r03srv05 | sdl-jupyterhub | c-vqbq2:p-t49mt |                60.0 
 2019-10-29 08:23:00.000 | jupyter-johan-2ekristiansson | p01r03srv10 | sdl-jupyterhub | c-vqbq2:p-t49mt |                60.0 
 2019-10-29 08:24:00.000 | jupyter-johan-2ekristiansson | p01r03srv10 | sdl-jupyterhub | c-vqbq2:p-t49mt |                60.0 


select 
  datasource_metering_pod_gpu_request.timestamp,
  datasource_metering_pod_gpu_request.labels['pod'] as pod, 
  datasource_metering_pod_gpu_request.labels['node'] as node, 
  datasource_metering_pod_gpu_request.labels['namespace'] as namespace, 
  datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] as projectid, 
  sum(datasource_metering_pod_gpu_request.amount * datasource_metering_pod_gpu_request.timeprecision) AS gpu_request_seconds 
  from datasource_metering_pod_gpu_request,datasource_metering_projectid_namespace_map 
  where datasource_metering_pod_gpu_request.timestamp=datasource_metering_projectid_namespace_map.timestamp 
  and datasource_metering_pod_gpu_request.labels['namespace'] LIKE datasource_metering_projectid_namespace_map.labels['namespace'] 
  group by datasource_metering_pod_gpu_request.timestamp,datasource_metering_pod_gpu_request.labels['pod'],datasource_metering_pod_gpu_request.labels['node'],datasource_metering_pod_gpu_request.labels['namespace'],datasource_metering_projectid_namespace_map.labels['annotation_field_cattle_io_projectId'] order by timestamp ASC LIMIT 5;

FIX presto-coordinator error:
Javacode gets nullpointerexception when trying to get username from local environment. Fixed by running following code on node running the container:
root@p01r03srv03:~# docker exec -it --user=root 6a78c678acdc 
"docker exec" requires at least 2 arguments.
See 'docker exec --help'.

Usage:  docker exec [OPTIONS] CONTAINER COMMAND [ARG...]

Run a command in a running container
root@p01r03srv03:~# docker exec -it --user=root 6a78c678acdc /bin/bash
bash-4.2# id
uid=0(root) gid=0(root) groups=0(root)
bash-4.2# groupadd --gid 1003 hive
bash-4.2# useradd --uid 1003 --gid hive --shell /bin/bash --home-dir /tmp hive
useradd: warning: the home directory already exists.
Not copying any file from skel directory into it.


